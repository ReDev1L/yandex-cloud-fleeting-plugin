package fleeting_plugin_yc

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1/instancegroup"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"log"
	"path"
)

var _ provider.InstanceGroup = (*InstanceGroup)(nil)

type InstanceGroup struct {
	FolderId           string `json:"folder_id"`
	Name               string `json:"name"`
	ConfigInstanceFile string `json:"config_file"`

	InstanceGroupId string

	size     int64
	ycsdk    *ycsdk.SDK
	settings provider.Settings

	logger hclog.Logger
}

func (g *InstanceGroup) Init(ctx context.Context, logger hclog.Logger, settings provider.Settings) (provider.ProviderInfo, error) {
	g.logger = logger.With("folder", g.FolderId, "name", g.Name)

	sdk, err := ycsdk.Build(ctx, ycsdk.Config{
		Credentials: ycsdk.InstanceServiceAccount(),
	})
	if err != nil {
		return provider.ProviderInfo{}, err
	}

	instancegroupTemplateNetwork := &instancegroup.NetworkInterfaceSpec{
		NetworkId:            "enp6hv3a5112ftavu61v",
		SubnetIds:            []string{"e9bs9blurio17qb374qo"},
		PrimaryV4AddressSpec: &instancegroup.PrimaryAddressSpec{},
	}

	var listInstancegroupTemplateNetwork []*instancegroup.NetworkInterfaceSpec
	var instancegroupTemplateZone []*instancegroup.AllocationPolicy_Zone

	listInstancegroupTemplateNetwork = append(listInstancegroupTemplateNetwork, instancegroupTemplateNetwork)
	instancegroupTemplateZone = append(instancegroupTemplateZone, &instancegroup.AllocationPolicy_Zone{ZoneId: "ru-central1-a"})

	instancegroupTemplate := &instancegroup.InstanceTemplate{
		Name:       "default-template-{instance.index}",
		PlatformId: "standard-v3",
		ResourcesSpec: &instancegroup.ResourcesSpec{
			Cores:        2,
			Memory:       2147483648,
			CoreFraction: 100,
		},
		BootDiskSpec: &instancegroup.AttachedDiskSpec{
			Mode: instancegroup.AttachedDiskSpec_READ_WRITE,
			DiskSpec: &instancegroup.AttachedDiskSpec_DiskSpec{
				Size:   17179869184,
				TypeId: "network-ssd",
				SourceOneof: &instancegroup.AttachedDiskSpec_DiskSpec_ImageId{
					ImageId: "fd81n0sfjm6d5nq6l05g",
				},
			},
		},
		NetworkInterfaceSpecs: listInstancegroupTemplateNetwork,
		SchedulingPolicy:      &instancegroup.SchedulingPolicy{Preemptible: true},
	}

	instanceGroupCreateReq := &instancegroup.CreateInstanceGroupRequest{
		FolderId:         "b1g4pju4m31dlfv6hut0",
		Name:             "test-instance-group-1",
		InstanceTemplate: instancegroupTemplate,
		ScalePolicy:      &instancegroup.ScalePolicy{ScaleType: &instancegroup.ScalePolicy_FixedScale_{FixedScale: &instancegroup.ScalePolicy_FixedScale{Size: 1}}},
		DeployPolicy: &instancegroup.DeployPolicy{
			MaxCreating:    1,
			MaxUnavailable: 1,
		},
		AllocationPolicy: &instancegroup.AllocationPolicy{Zones: instancegroupTemplateZone},
		ServiceAccountId: "ajegtcbjqc69q6re14sk",
	}

	instancegroupListReq := instancegroup.ListInstanceGroupsRequest{FolderId: "b1g4pju4m31dlfv6hut0"}

	instancegroupListResp, err := sdk.InstanceGroup().InstanceGroup().List(ctx, &instancegroupListReq)

	for _, ig := range instancegroupListResp.InstanceGroups {
		if ig.Name == g.Name {
			g.InstanceGroupId = ig.Id
			g.size = ig.ScalePolicy.GetFixedScale().Size
		}
	}

	if len(g.InstanceGroupId) < 1 {
		op, err := sdk.WrapOperation(sdk.InstanceGroup().InstanceGroup().Create(ctx, instanceGroupCreateReq))

		if err != nil {
			log.Fatal(err)
		}
		err = op.Wait(ctx)
		if err != nil {
			log.Fatal(err)
		}

		response, _ := op.Response()
		proto := response.ProtoReflect()
		instanceGroupId := response.ProtoReflect().Get(proto.Descriptor().Fields().ByName("id"))

		g.InstanceGroupId = instanceGroupId.String()
		g.size = 1
	}

	g.ycsdk = sdk
	g.settings = settings

	return provider.ProviderInfo{
		ID:      path.Join("yc", g.FolderId, g.Name),
		MaxSize: 100,
		Version: "0.0.1",
	}, nil

}

func (g *InstanceGroup) Update(ctx context.Context, update func(instance string, state provider.State)) error {
	instances, err := g.ycsdk.InstanceGroup().InstanceGroup().ListInstances(ctx,
		&instancegroup.ListInstanceGroupInstancesRequest{
			InstanceGroupId: g.InstanceGroupId,
		})
	for _, instance := range instances.Instances {
		var state provider.State
		switch instance.Status.String() {
		case "PREPARING_RESOURCES", "CREATING_INSTANCE", "STARTING_INSTANCE":
			state = provider.StateCreating
		case "DELETING_INSTANCE", "STOPPING_INSTANCE", "DELETED":
			state = provider.StateDeleting
		case "RUNNING_ACTUAL":
			state = provider.StateRunning
		default:
			g.logger.Warn("unknown instance status", "instance", instance.Name, "status", instance.Status.String())
			continue
		}

		update(instance.Id, state)
	}

	if err != nil {
		return err
	}

	return nil
}

func (g *InstanceGroup) Increase(ctx context.Context, delta int) (succeeded int, err error) {

	newSizeInstanceGroup := g.size + int64(delta)
	increaseInstanceReq := instancegroup.UpdateInstanceGroupRequest{
		InstanceGroupId: g.InstanceGroupId,
		ScalePolicy:     &instancegroup.ScalePolicy{ScaleType: &instancegroup.ScalePolicy_FixedScale_{FixedScale: &instancegroup.ScalePolicy_FixedScale{Size: newSizeInstanceGroup}}},
		UpdateMask:      &fieldmaskpb.FieldMask{Paths: []string{"scale_policy"}},
	}
	op, err := g.ycsdk.WrapOperation(g.ycsdk.InstanceGroup().InstanceGroup().Update(ctx, &increaseInstanceReq))

	if err != nil {
		log.Fatal(err)
		return 0, err
	}

	err = op.Wait(ctx)
	if err != nil {
		log.Fatal(err)
		return 0, err
	}

	return delta, nil
}

func (g *InstanceGroup) Decrease(ctx context.Context, instances []string) (succeeded []string, err error) {
	instanceGroupDeleteReq := instancegroup.DeleteInstancesRequest{
		InstanceGroupId:    g.InstanceGroupId,
		ManagedInstanceIds: instances,
	}

	op, err := g.ycsdk.WrapOperation(g.ycsdk.InstanceGroup().InstanceGroup().DeleteInstances(ctx, &instanceGroupDeleteReq))

	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	err = op.Wait(ctx)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	g.size = g.size - int64(len(instances))
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
			info.ExternalAddr = ipAddress.OneToOneNat.Address
		}
	}

	return info, nil
}

func (g *InstanceGroup) Shutdown(ctx context.Context) error {
	//TODO implement me
	return nil
}
