package storage

// import (
// 	"fmt"

// 	"github.com/libopenstorage/cloudops"
// 	"github.com/libopenstorage/cloudops/api/client"
// )

// // // Create a new Vol for the specific volume spev.c.
// // // It returns a system generated VolumeID that uniquely identifies the volume
// // func (v *volumeClient) Create(ctx context.Context, locator *api.VolumeLocator, source *api.Source,
// // 	spec *api.VolumeSpec) (string, error) {
// // 	response := &api.VolumeCreateResponse{}
// // 	request := &api.VolumeCreateRequest{
// // 		Locator: locator,
// // 		Source:  source,
// // 		Spec:    spec,
// // 	}
// // 	if err := v.c.Post().Resource(volumePath).Body(request).Do().Unmarshal(response); err != nil {
// // 		return "", err
// // 	}
// // 	if response.VolumeResponse != nil && response.VolumeResponse.Error != "" {
// // 		return "", errors.New(response.VolumeResponse.Error)
// // 	}
// // 	return response.Id, nil
// // }

// // // Delete volume.
// // // Errors ErrEnoEnt, ErrVolHasSnaps may be returned.
// // func (v *volumeClient) Delete(ctx context.Context, volumeID string) error {
// // 	response := &api.VolumeResponse{}
// // 	if err := v.c.Delete().Resource(volumePath).Instance(volumeID).Do().Unmarshal(response); err != nil {
// // 		return err
// // 	}
// // 	if response.Error != "" {
// // 		return errors.New(response.Error)
// // 	}
// // 	return nil
// // }

// // NewDriverClient returns a new REST client of the supplied version for specified driver.
// // host: REST endpoint [http://<ip>:<port> OR unix://<path-to-unix-socket>]. default: [unix:///var/lib/osd/<driverName>.sock]
// // version: Volume API version
// // userAgent: Drivername for http connections
// func NewDriverClient(host, driverName, version, userAgent string) (*client.Client, error) {
// 	if host == "" {
// 		if driverName == "" {
// 			return nil, fmt.Errorf("driver Name cannot be empty")
// 		}
// 		host = client.GetUnixServerPath(driverName, "/var/lib/osd/cloudops/")
// 	}

// 	return client.NewClient(host, version, userAgent)
// }

// type storageDriver struct {
// 	storage cloudops.Storage
// }

// type volumeClient struct {
// 	c *client.Client
// }

// func newVolumeClient(c *client.Client) cloudops.Storage {
// 	return &storageDriver{
// 		c: c}
// }
