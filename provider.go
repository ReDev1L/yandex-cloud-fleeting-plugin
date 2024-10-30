package fleeting_plugin_yc

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"

	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-hclog"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1/instancegroup"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

var _ provider.InstanceGroup = (*InstanceGroup)(nil)

var charset = []byte("abcdefghijklmnopqrstuvwxyz")

func randomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		// randomly select 1 character from given charset
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

type InstanceGroup struct {
	FolderId           string `json:"folder_id"`
	Name               string `json:"name"`
	ConfigInstanceFile string `json:"config_file"`
	SshPrivateKeyFile  string `json:"ssh_file,omitempty"`

	SshPublicKey    string
	InstanceGroupId string

	size     int64
	ycsdk    *ycsdk.SDK
	settings provider.Settings

	template InstanceTemplate

	logger hclog.Logger
}

func (g *InstanceGroup) GenerateInstanceGroup() *instancegroup.InstanceTemplate {
	instanceGroupTemplateNetwork := &instancegroup.NetworkInterfaceSpec{
		NetworkId:            g.template.Network.Network,
		SubnetIds:            []string{g.template.Network.Subnet},
		PrimaryV4AddressSpec: &instancegroup.PrimaryAddressSpec{},
	}

	if g.template.Network.Nat {
		instanceGroupTemplateNetwork.PrimaryV4AddressSpec = &instancegroup.PrimaryAddressSpec{OneToOneNatSpec: &instancegroup.OneToOneNatSpec{IpVersion: instancegroup.IpVersion_IPV4}}
	}

	if len(g.template.Network.Security) > 0 {
		instanceGroupTemplateNetwork.SecurityGroupIds = []string{g.template.Network.Security}
	}

	var listInstanceGroupTemplateNetwork []*instancegroup.NetworkInterfaceSpec

	listInstanceGroupTemplateNetwork = append(listInstanceGroupTemplateNetwork, instanceGroupTemplateNetwork)

	instanceGroupTemplate := &instancegroup.InstanceTemplate{
		Name:       g.Name + "-runner-" + randomString(7) + "-" + "{instance.index}",
		Labels:     g.GenerateLabels(),
		PlatformId: g.template.Platform,
		ResourcesSpec: &instancegroup.ResourcesSpec{
			Cores:        g.template.Resources.CPU,
			Memory:       g.template.Resources.Memory,
			CoreFraction: g.template.Resources.Fraction,
		},
		BootDiskSpec: &instancegroup.AttachedDiskSpec{
			Mode: instancegroup.AttachedDiskSpec_READ_WRITE,
			DiskSpec: &instancegroup.AttachedDiskSpec_DiskSpec{
				Size:   g.template.Disk.Size,
				TypeId: g.template.Disk.Type,
				SourceOneof: &instancegroup.AttachedDiskSpec_DiskSpec_ImageId{
					ImageId: g.template.Disk.Image,
				},
			},
		},
		NetworkInterfaceSpecs: listInstanceGroupTemplateNetwork,
		SchedulingPolicy:      &instancegroup.SchedulingPolicy{Preemptible: true},
		Metadata:              g.GenerateMeta(),
	}

	if g.template.VmServiceAccount != "" {
		instanceGroupTemplate.ServiceAccountId = g.template.VmServiceAccount
	}

	return instanceGroupTemplate
}

func (g *InstanceGroup) GenerateInstanceGroupRequest() *instancegroup.CreateInstanceGroupRequest {

	instanceTemplate := g.GenerateInstanceGroup()
	var instanceGroupTemplateZone []*instancegroup.AllocationPolicy_Zone
	instanceGroupTemplateZone = append(instanceGroupTemplateZone, &instancegroup.AllocationPolicy_Zone{ZoneId: g.template.Zone})
	instanceGroupCreateReq := &instancegroup.CreateInstanceGroupRequest{
		FolderId:         g.FolderId,
		Name:             g.Name,
		InstanceTemplate: instanceTemplate,
		ScalePolicy:      &instancegroup.ScalePolicy{ScaleType: &instancegroup.ScalePolicy_FixedScale_{FixedScale: &instancegroup.ScalePolicy_FixedScale{Size: 0}}},
		DeployPolicy: &instancegroup.DeployPolicy{
			MaxCreating:    3,
			MaxUnavailable: 1,
			MaxDeleting:    1,
		},
		AllocationPolicy: &instancegroup.AllocationPolicy{Zones: instanceGroupTemplateZone},
		ServiceAccountId: g.template.ServiceAccount,
	}

	return instanceGroupCreateReq
}

func (g *InstanceGroup) GenerateUpdateInstanceGroupRequest() *instancegroup.UpdateInstanceGroupRequest {
	instanceTemplate := g.GenerateInstanceGroup()
	var instanceGroupTemplateZone []*instancegroup.AllocationPolicy_Zone
	instanceGroupTemplateZone = append(instanceGroupTemplateZone, &instancegroup.AllocationPolicy_Zone{ZoneId: g.template.Zone})
	instanceGroupCreateReq := &instancegroup.UpdateInstanceGroupRequest{
		InstanceGroupId:  g.InstanceGroupId,
		Name:             g.Name,
		InstanceTemplate: instanceTemplate,
		ScalePolicy:      &instancegroup.ScalePolicy{ScaleType: &instancegroup.ScalePolicy_FixedScale_{FixedScale: &instancegroup.ScalePolicy_FixedScale{Size: 0}}},
		DeployPolicy: &instancegroup.DeployPolicy{
			MaxCreating:    3,
			MaxUnavailable: 1,
			MaxDeleting:    1,
		},
		AllocationPolicy: &instancegroup.AllocationPolicy{Zones: instanceGroupTemplateZone},
		ServiceAccountId: g.template.ServiceAccount,
	}

	return instanceGroupCreateReq

}

func (g *InstanceGroup) GenerateMeta() map[string]string {
	var iTemplate InstanceTemplate

	metaData := make(map[string]string)

	metaData["ssh-keys"] = g.settings.Username + ":" + g.SshPublicKey

	ymlData, err := os.ReadFile(g.ConfigInstanceFile)

	if err != nil {
		return metaData
	}

	err = yaml.Unmarshal(ymlData, &iTemplate)

	if err != nil {
		return metaData
	}

	if iTemplate.Monitoring.Enabled {
		metaData["install-unified-agent"] = "1"
		metaData["user-data"] = "#cloud-config\nruncmd:\n  - wget -O - https://monitoring.api.cloud.yandex.net/monitoring/v2/unifiedAgent/config/install.sh | bash"
	}

	return metaData
}

func (g *InstanceGroup) GenerateLabels() map[string]string {

	labels := make(map[string]string)

	labels["fleet-runner"] = "true"
	labels["fleet-runner-name"] = g.Name

	return labels
}

func (g *InstanceGroup) Init(ctx context.Context, logger hclog.Logger, settings provider.Settings) (provider.ProviderInfo, error) {
	sdk, err := ycsdk.Build(ctx, ycsdk.Config{
		Credentials: ycsdk.InstanceServiceAccount(),
	})

	if err != nil {
		log.Fatal(err)
	}

	g.settings = settings
	log.Println("Running with settings: ", g.settings)
	pemBytes, err := os.ReadFile(g.SshPrivateKeyFile)

	if err != nil {
		log.Println("Not set private SSH file")
	}

	var iTemplate InstanceTemplate
	ymlData, err := os.ReadFile(g.ConfigInstanceFile)

	if err != nil {
		log.Fatal("Error in get Template ", err)
	}

	err = yaml.Unmarshal(ymlData, &iTemplate)

	if err != nil {
		log.Fatal("Error in get Template ", err)
	}

	g.template = iTemplate

	g.settings.Key = pemBytes
	g.size = 0

	info := provider.ConnectInfo{ConnectorConfig: g.settings.ConnectorConfig}

	err = g.ssh(&info)

	if err != nil {
		return provider.ProviderInfo{}, err
	}

	instanceGroupListResp, err := sdk.InstanceGroup().InstanceGroup().List(ctx, &instancegroup.ListInstanceGroupsRequest{FolderId: g.FolderId})

	if err != nil {
		log.Fatal("Error in get Instance group ", err)
	}

	for _, ig := range instanceGroupListResp.InstanceGroups {
		if ig.Name == g.Name {
			g.InstanceGroupId = ig.Id
		}
	}

	if len(g.InstanceGroupId) < 1 {
		op, err := sdk.WrapOperation(sdk.InstanceGroup().InstanceGroup().Create(ctx, g.GenerateInstanceGroupRequest()))

		if err != nil {
			log.Fatal("Creating instance group ", err)
		}

		err = op.Wait(ctx)
		if err != nil {
			log.Fatal("Created instance group ", err)
		}

		response, _ := op.Response()
		proto := response.ProtoReflect()
		instanceGroupId := response.ProtoReflect().Get(proto.Descriptor().Fields().ByName("id"))

		g.InstanceGroupId = instanceGroupId.String()
	} else {
		log.Println("Use exist IG with ID", g.InstanceGroupId, "name", g.Name, "current size", g.size)
		op, err := sdk.WrapOperation(sdk.InstanceGroup().InstanceGroup().Update(ctx, g.GenerateUpdateInstanceGroupRequest()))
		log.Println("IG updated", g.InstanceGroupId, g.Name)
		if err != nil {
			log.Fatal("Updating instance group failed", err)
		}

		err = op.Wait(ctx)
		if err != nil {
			log.Fatal("Updated instance group failed", err)
		}
	}

	g.ycsdk = sdk

	return provider.ProviderInfo{
		ID:      path.Join("yc", g.FolderId, g.Name),
		MaxSize: 100,
		Version: "1.1.0",
	}, nil

}

func (g *InstanceGroup) Update(ctx context.Context, update func(instance string, state provider.State)) error {
	instances, err := g.ycsdk.InstanceGroup().InstanceGroup().ListInstances(ctx,
		&instancegroup.ListInstanceGroupInstancesRequest{
			InstanceGroupId: g.InstanceGroupId,
		})

	if err != nil {
		return err
	}

	for _, instance := range instances.Instances {
		state := provider.StateCreating
		switch instance.Status.String() {
		case "PREPARING_RESOURCES", "CREATING_INSTANCE", "STARTING_INSTANCE":
			state = provider.StateCreating
		case "DELETING_INSTANCE", "STOPPING_INSTANCE":
			state = provider.StateDeleting
		case "DELETED":
			state = provider.StateDeleted
		case "RUNNING_ACTUAL":
			state = provider.StateRunning
		}
		update(instance.Id, state)
	}

	return nil
}

func (g *InstanceGroup) Increase(ctx context.Context, delta int) (succeeded int, err error) {
	var updateMaskPath []string
	log.Println("Increase on ", delta, " current IG size", g.size)
	updateMaskPath = append(updateMaskPath, "instance_template.metadata")
	updateMaskPath = append(updateMaskPath, "scale_policy")
	newSizeInstanceGroup := g.size + int64(delta)

	increaseInstanceReq := instancegroup.UpdateInstanceGroupRequest{
		InstanceGroupId:  g.InstanceGroupId,
		InstanceTemplate: &instancegroup.InstanceTemplate{Metadata: g.GenerateMeta()},
		ScalePolicy:      &instancegroup.ScalePolicy{ScaleType: &instancegroup.ScalePolicy_FixedScale_{FixedScale: &instancegroup.ScalePolicy_FixedScale{Size: newSizeInstanceGroup}}},
		UpdateMask:       &fieldmaskpb.FieldMask{Paths: updateMaskPath},
	}

	op, err := g.ycsdk.WrapOperation(g.ycsdk.InstanceGroup().InstanceGroup().Update(ctx, &increaseInstanceReq))

	if err != nil {
		log.Println(err)
		return 0, err
	}

	err = op.Wait(ctx)
	if err != nil {
		log.Println(err)
		return 0, err
	}

	g.size = newSizeInstanceGroup
	log.Println("Increased IG new size: ", g.size)
	return delta, nil
}

func (g *InstanceGroup) Decrease(ctx context.Context, instances []string) (succeeded []string, err error) {

	instanceGroupDeleteReq := instancegroup.DeleteInstancesRequest{
		InstanceGroupId:    g.InstanceGroupId,
		ManagedInstanceIds: instances,
	}

	op, err := g.ycsdk.WrapOperation(g.ycsdk.InstanceGroup().InstanceGroup().DeleteInstances(ctx, &instanceGroupDeleteReq))

	if err != nil {
		log.Println(err)
		return nil, err
	}
	err = op.Wait(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	log.Println("Decrease on", len(instances), "current IG size", g.size)
	g.size = g.size - int64(len(instances))
	if g.size < 0 {
		g.size = 0
	}
	log.Println("Decreased new IG size", g.size)
	return instances, err

}

func (g *InstanceGroup) ConnectInfo(ctx context.Context, instanceId string) (provider.ConnectInfo, error) {
	info := provider.ConnectInfo{ConnectorConfig: g.settings.ConnectorConfig}

	instances, err := g.ycsdk.InstanceGroup().InstanceGroup().ListInstances(ctx,
		&instancegroup.ListInstanceGroupInstancesRequest{
			InstanceGroupId: g.InstanceGroupId,
		})
	if err != nil {
		return provider.ConnectInfo{}, err
	}
	for _, instance := range instances.Instances {
		if instance.Id == instanceId {
			if instance.Status.String() != "RUNNING_ACTUAL" {
				return provider.ConnectInfo{}, fmt.Errorf("instance status is not running (%s)", instance.Status.String())
			}
			ipAddress := instance.NetworkInterfaces[0].GetPrimaryV4Address()

			info.InternalAddr = ipAddress.Address
			info.ExternalAddr = ipAddress.OneToOneNat.GetAddress()
		}
	}
	log.Println("Connecting to instance ", instanceId, "with IP address: ", info.ExternalAddr, info.InternalAddr)
	return info, nil
}

func (g *InstanceGroup) Shutdown(ctx context.Context) error {

	increaseInstanceReq := instancegroup.UpdateInstanceGroupRequest{
		InstanceGroupId: g.InstanceGroupId,
		ScalePolicy:     &instancegroup.ScalePolicy{ScaleType: &instancegroup.ScalePolicy_FixedScale_{FixedScale: &instancegroup.ScalePolicy_FixedScale{Size: 0}}},
		UpdateMask:      &fieldmaskpb.FieldMask{Paths: []string{"scale_policy"}},
	}

	op, err := g.ycsdk.WrapOperation(g.ycsdk.InstanceGroup().InstanceGroup().Update(ctx, &increaseInstanceReq))

	if err != nil {
		log.Println(err)
		return err
	}

	err = op.Wait(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
