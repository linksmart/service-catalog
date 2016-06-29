package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"linksmart.eu/lc/core/catalog"
	"linksmart.eu/lc/sec/auth/obtainer"
	"strings"
)

type RemoteCatalogClient struct {
	serverEndpoint *url.URL
	ticket         *obtainer.Client
}

func NewRemoteCatalogClient(serverEndpoint string, ticket *obtainer.Client) CatalogClient {
	// Check if serverEndpoint is a correct URL
	endpointUrl, err := url.Parse(serverEndpoint)
	if err != nil {
		return &RemoteCatalogClient{}
	}

	return &RemoteCatalogClient{
		serverEndpoint: endpointUrl,
		ticket:         ticket,
	}
}

// Retrieves a device
func (c *RemoteCatalogClient) Get(id string) (*SimpleDevice, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v/%v", c.serverEndpoint, TypeDevices, id),
		nil,
		nil,
		c.ticket,
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return nil, &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return nil, &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return nil, &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Error getting device: %v", ErrorMsg(res))
		}
	}

	decoder := json.NewDecoder(res.Body)
	var d SimpleDevice
	err = decoder.Decode(&d)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

// Adds a device and returns its id
func (c *RemoteCatalogClient) Add(d *Device) (string, error) {
	device := *d
	id := device.Id
	device.Id = ""
	b, _ := json.Marshal(device)

	var (
		res *http.Response
		err error
	)

	if id == "" { // Let the system generate an id
		res, err = catalog.HTTPRequest("POST",
			fmt.Sprintf("%v/%v/", c.serverEndpoint, TypeDevices),
			map[string][]string{"Content-Type": []string{"application/ld+json"}},
			bytes.NewReader(b),
			c.ticket,
		)
		if err != nil {
			return "", err
		}
		defer res.Body.Close()

	} else { // User-defined id

		// Check if id is unique
		res, err = catalog.HTTPRequest("GET",
			fmt.Sprintf("%v/%v/%v", c.serverEndpoint, TypeDevices, id),
			nil,
			nil,
			c.ticket,
		)
		if err != nil {
			return "", err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			return "", &ConflictError{fmt.Sprintf("Device id %s is not unique.", id)}
		}

		// Now add
		res, err = catalog.HTTPRequest("PUT",
			fmt.Sprintf("%v/%v/%v", c.serverEndpoint, TypeDevices, id),
			map[string][]string{"Content-Type": []string{"application/ld+json"}},
			bytes.NewReader(b),
			c.ticket,
		)
		if err != nil {
			return "", err
		}
		defer res.Body.Close()
	}

	switch res.StatusCode {
	case http.StatusBadRequest:
		return "", &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return "", &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return "", &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusCreated {
			return "", fmt.Errorf("Error adding device: %v", ErrorMsg(res))
		}
	}

	location, err := res.Location()
	if err != nil {
		return "", err
	}
	id = strings.SplitAfter(location.String(), TypeDevices+"/")[1]

	return id, nil
}

// Updates a device
func (c *RemoteCatalogClient) Update(id string, d *Device) error {
	// Check if id is found
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v/%v", c.serverEndpoint, TypeDevices, id),
		nil,
		nil,
		c.ticket,
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return &NotFoundError{fmt.Sprintf("Device with id %s is not found.", id)}
	}

	b, _ := json.Marshal(d)
	res, err = catalog.HTTPRequest("PUT",
		fmt.Sprintf("%v/%v/%v", c.serverEndpoint, TypeDevices, id),
		map[string][]string{"Content-Type": []string{"application/ld+json"}},
		bytes.NewReader(b),
		c.ticket,
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("Error updating device: %v", ErrorMsg(res))
		}
	}

	return nil
}

// Deletes a device
func (c *RemoteCatalogClient) Delete(id string) error {
	res, err := catalog.HTTPRequest("DELETE",
		fmt.Sprintf("%v/%v/%v", c.serverEndpoint, TypeDevices, id),
		nil,
		bytes.NewReader([]byte{}),
		c.ticket,
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("Error deleting device: %v", ErrorMsg(res))
		}
	}

	return nil
}

// Retrieves a page from the device collection
func (c *RemoteCatalogClient) List(page int, perPage int) ([]SimpleDevice, int, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v?%v=%v&%v=%v", c.serverEndpoint, TypeDevices,
			catalog.GetParamPage, page, catalog.GetParamPerPage, perPage),
		nil,
		nil,
		c.ticket,
	)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return nil, 0, &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return nil, 0, &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return nil, 0, &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return nil, 0, fmt.Errorf("Error listing devices: %v", ErrorMsg(res))
		}
	}

	decoder := json.NewDecoder(res.Body)
	var coll DeviceCollection
	err = decoder.Decode(&coll)
	if err != nil {
		return nil, 0, err
	}

	return coll.Devices, coll.Total, nil
}

// Filters devices
func (c *RemoteCatalogClient) Filter(path, op, value string, page, perPage int) ([]SimpleDevice, int, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v/%v/%v/%v?%v=%v&%v=%v",
			c.serverEndpoint, TypeDevices, path, op, value,
			catalog.GetParamPage, page, catalog.GetParamPerPage, perPage),
		nil,
		nil,
		c.ticket,
	)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return nil, 0, &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return nil, 0, &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return nil, 0, &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return nil, 0, fmt.Errorf("Error filtering devices: %v", ErrorMsg(res))
		}
	}

	decoder := json.NewDecoder(res.Body)
	var coll DeviceCollection
	err = decoder.Decode(&coll)
	if err != nil {
		return nil, 0, err
	}

	return coll.Devices, coll.Total, nil
}

// Retrieves a resource
func (c *RemoteCatalogClient) GetResource(id string) (*Resource, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v/%v", c.serverEndpoint, TypeResources, id),
		nil,
		nil,
		c.ticket,
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return nil, &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return nil, &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return nil, &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Error getting resource: %v", ErrorMsg(res))
		}
	}

	decoder := json.NewDecoder(res.Body)
	var r Resource
	err = decoder.Decode(&r)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// Retrieves a page from the resource collection
func (c *RemoteCatalogClient) ListResources(page int, perPage int) ([]Resource, int, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v?%v=%v&%v=%v",
			c.serverEndpoint, TypeResources,
			catalog.GetParamPage, page, catalog.GetParamPerPage, perPage),
		nil,
		nil,
		c.ticket,
	)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return nil, 0, &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return nil, 0, &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return nil, 0, &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return nil, 0, fmt.Errorf("Error listing resources: %v", ErrorMsg(res))
		}
	}

	decoder := json.NewDecoder(res.Body)
	var coll ResourceCollection
	err = decoder.Decode(&coll)
	if err != nil {
		return nil, 0, err
	}

	return coll.Resources, coll.Total, nil
}

// Filter resources
func (c *RemoteCatalogClient) FilterResources(path, op, value string, page, perPage int) ([]Resource, int, error) {
	res, err := catalog.HTTPRequest("GET",
		fmt.Sprintf("%v/%v/%v/%v/%v?%v=%v&%v=%v",
			c.serverEndpoint, TypeResources, path, op, value,
			catalog.GetParamPage, page, catalog.GetParamPerPage, perPage),
		nil,
		nil,
		c.ticket,
	)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusBadRequest:
		return nil, 0, &BadRequestError{ErrorMsg(res)}
	case http.StatusConflict:
		return nil, 0, &ConflictError{ErrorMsg(res)}
	case http.StatusNotFound:
		return nil, 0, &NotFoundError{ErrorMsg(res)}
	default:
		if res.StatusCode != http.StatusOK {
			return nil, 0, fmt.Errorf("Error filtering resources: %v", ErrorMsg(res))
		}
	}

	decoder := json.NewDecoder(res.Body)
	var coll ResourceCollection
	err = decoder.Decode(&coll)
	if err != nil {
		return nil, 0, err
	}

	return coll.Resources, coll.Total, nil
}

// Returns the message field of a resource.Error response
func ErrorMsg(res *http.Response) string {
	decoder := json.NewDecoder(res.Body)

	var e Error
	err := decoder.Decode(&e)
	if err != nil {
		return res.Status
	}
	return e.Message
}
