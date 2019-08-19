## How to test

### Setup environment variables

You will first need to have a vSphere environment and then provide details of the vcenter server and VM as below.

```bash
export VSPHERE_VCENTER=my-vc.demo.com
export VSPHERE_VCENTER_PORT=443
export VSPHERE_USER=administrator@vsphere.local
export VSPHERE_PASSWORD=my-vc-password
export VSPHERE_INSECURE=true
export VSPHERE_TEST_DATASTORE=Phy-vsanDatastore
export VSPHERE_TEST_HOST=70.0.0.162
export VSPHERE_VM_UUID=<vm-uuid>
```

`VSPHERE_TEST_DATASTORE` above can be a vSphere datastore or datastore cluster name. When testing changes, it is recommended to test with both.

### To get VSPHERE_VM_UUID,


#### If in a Kubernetes cluster,

Do a `kubectl get nodes <node-name> -o yaml`. The spec.providerID will have the VM's UUID.

In below example, 4223719d-c9c7-c8a9-0fb9-25f4f2a66fd4 is the UUID

```
spec:
    providerID: vsphere://4223719d-c9c7-c8a9-0fb9-25f4f2a66fd4
```

#### If not in a  Kubernetes cluster,

Go to https://<vcenter-ip>/mob/?moid=<VM-MOREF>&doPath=config and get the "uuid" field value

To get the VM-MOREF, select the VM in vcenter server and you will see a string of format "VirtualMachine:vm-155" in the URL. vm-155 is the moref.

```
export VSPHERE_VM_UUID=42124a20-d049-9c0a-0094-1552b320fb18
export VSPHERE_TEST_DATASTORE=<test-datastore-to-use>
```

### Running the test


To run all tests,
```bash
go test -v
```

To run a particular test (e.g DeviceMappings)

```bash
go test -v -run TestDeviceMappings
```

To skip storage tests and only run compute tests

```bash
go test -v -args -skipStorageTests
```

