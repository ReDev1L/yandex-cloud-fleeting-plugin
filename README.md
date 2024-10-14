# Fleeting Plugin Yandex Cloud

This is a [fleeting](https://gitlab.com/gitlab-org/fleeting/fleeting) plugin for Yandex Cloud.

## Plugin Configuration

The following parameters are supported:

| Parameter             | Type   | Description                                           |
|-----------------------|--------|-------------------------------------------------------|
| `name`                | string | Name of the Auto Scaling Group                        |
| `folder_id`           | string | ID Folder in YC where running autoscale gitlab-runner |
| `config_file`         | string | Path to YML file for instance configuration           |
| `ssh_file`            | string | Path to Private ssh-key *optional                     |

## Template Example

```
platform: "standard-v3"
zone: "ru-central1-a"
preemptible: true
serviceAccount: ""

resources:
  cpu: 2
  memory: 2147483648
  fraction: 100

disk:
  type: "network-ssd"
  size: 17179869184
  image: ""

network:
  network: ""
  subnet: ""
  security: ""
  nat: true

```

**serviceAccount** must have compute.admin, vpc.admin

The plugin uses a service account that is attached to the VM, which must have **_admin_** rights