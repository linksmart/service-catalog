// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"code.linksmart.eu/sc/service-catalog/utils"
	"github.com/satori/go.uuid"
)

func setup() (*Controller, func(), error) {
	var (
		storage Storage
		err     error
		tempDir string = fmt.Sprintf("%s/lslc/test-%s.ldb", strings.Replace(os.TempDir(), "\\", "/", -1), uuid.NewV4().String())
	)
	switch TestStorageType {
	case CatalogBackendMemory:
		storage = NewMemoryStorage()
	case CatalogBackendLevelDB:
		storage, err = NewLevelDBStorage(tempDir, nil)
		if err != nil {
			return nil, nil, err
		}
	}

	controller, err := NewController(storage)
	if err != nil {
		storage.Close()
		return nil, nil, err
	}

	return controller, func() {
		controller.Stop()
		os.RemoveAll(tempDir) // Remove temp files
	}, nil
}

func TestAddService(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	// User-defined id
	var r Service
	r.ID = "E9203BE9-D705-42A8-8B12-F28E7EA2FC99"
	r.Name = "_test._tcp"
	r.TTL = 30

	s, err := controller.add(r)
	if err != nil {
		t.Fatalf("Unexpected error on add: %v", err.Error())
	}
	if s.ID != r.ID {
		t.Fatalf("User defined ID is not returned. Getting %v instead of %v\n", s.ID, r.ID)
	}

	_, err = controller.add(r)
	if err == nil {
		t.Error("Didn't get any error when adding a service with non-unique id.")
	}

	// System-generated id
	var r2 Service
	r2.Name = "_test._tcp"
	s, err = controller.add(r2)
	if err != nil {
		t.Fatalf("Unexpected error on add: %v", err.Error())
	}
	if !strings.ContainsAny(s.ID, "-") {
		t.Fatalf("System-generated ID does not look like a UUID. Getting location: %v\n", s.ID)
	}
}

func TestUpdateService(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	var r Service
	r.ID = "E9203BE9-D705-42A8-8B12-F28E7EA2FC99"
	r.Name = "_test._tcp"
	r.TTL = 30

	_, err = controller.add(r)
	if err != nil {
		t.Errorf("Unexpected error on add: %v", err.Error())
	}
	r.Description = "new description"

	_, err = controller.update(r.ID, r)
	if err != nil {
		t.Errorf("Unexpected error on update: %v", err.Error())
	}

	rg, err := controller.get(r.ID)
	if err != nil {
		t.Error("Unexpected error on get: %v", err.Error())
	}

	if rg.Description != r.Description {
		t.Fail()
	}
}

func TestGetService(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	var r Service
	r.Description = "some description"
	r.ID = "E9203BE9-D705-42A8-8B12-F28E7EA2FC99"
	r.Name = "_test._tcp"
	r.TTL = 30

	_, err = controller.add(r)
	if err != nil {
		t.Errorf("Unexpected error on add: %v", err.Error())
	}

	rg, err := controller.get(r.ID)
	if err != nil {
		t.Error("Unexpected error on get: %v", err.Error())
	}

	if rg.ID != r.ID || rg.Description != r.Description || rg.TTL != r.TTL {
		t.Fail()
	}
}

func TestDeleteService(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	var r Service
	r.ID = "E9203BE9-D705-42A8-8B12-F28E7EA2FC99"
	r.Name = "_test._tcp"
	r.TTL = 30

	_, err = controller.add(r)
	if err != nil {
		t.Errorf("Unexpected error on add: %v", err.Error())
	}

	err = controller.delete(r.ID)
	if err != nil {
		t.Error("Unexpected error on delete: %v", err.Error())
	}

	err = controller.delete(r.ID)
	if err == nil {
		t.Error("Didn't get any error when deleting a deleted service.")
	}
}

func TestListServices(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	var r Service
	r.Name = "_test._tcp"

	// Add 10 entries
	for i := 0; i < 11; i++ {
		r.ID = "TestID_" + string(i)
		r.TTL = 30
		_, err := controller.add(r)

		if err != nil {
			t.Errorf("Unexpected error on add: %v", err.Error())
		}
	}

	p1pp2, total, _ := controller.list(1, 2)
	if total != 11 {
		t.Errorf("Expected total is 11, returned: %v", total)
	}

	if len(p1pp2) != 2 {
		t.Errorf("Wrong number of entries: requested page=1 , perPage=2. Expected: 2, returned: %v", len(p1pp2))
	}

	p2pp2, _, _ := controller.list(2, 2)
	if len(p2pp2) != 2 {
		t.Errorf("Wrong number of entries: requested page=2 , perPage=2. Expected: 2, returned: %v", len(p2pp2))
	}

	p2pp5, _, _ := controller.list(2, 5)
	if len(p2pp5) != 5 {
		t.Errorf("Wrong number of entries: requested page=2 , perPage=5. Expected: 5, returned: %v", len(p2pp5))
	}

	p4pp3, _, _ := controller.list(4, 3)
	if len(p4pp3) != 2 {
		t.Errorf("Wrong number of entries: requested page=4 , perPage=3. Expected: 2, returned: %v", len(p4pp3))
	}
}

func TestFilterService(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	for i := 0; i < 5; i++ {
		_, err := controller.add(Service{
			Description: fmt.Sprintf("boring_%d", i),
			Name:        "_test._tcp",
		})
		if err != nil {
			t.Fatal("Error adding a service:", err.Error())
		}
	}

	controller.add(Service{
		Description: "interesting_1",
		Name:        "_test._tcp",
	})
	controller.add(Service{
		Description: "interesting_2",
		Name:        "_test._tcp",
	})

	services, total, err := controller.filter("description", utils.FOpPrefix, "interesting", 1, 10)
	if err != nil {
		t.Fatal("Error filtering services:", err.Error())
	}
	if total != 2 {
		t.Fatalf("Returned %d instead of 2 services when filtering name/prefix/interesting: \n%v", total, services)
	}
	for _, s := range services {
		if !strings.Contains(s.Description, "interesting") {
			t.Fatal("Wrong results when filtering name/prefix/interesting:\n", s)
		}
	}
}

func TestCleanExpired(t *testing.T) {
	t.Log(TestStorageType)
	controller, shutdown, err := setup()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer shutdown()

	var d = Service{
		Description: "my_service",
		Name:        "_test._tcp",
		TTL:         1,
	}

	s, err := controller.add(d)
	if err != nil {
		t.Fatal("Error adding a service:", err.Error())
	}

	addingTime := time.Now()
	time.Sleep(6 * time.Second)

	checkingTime := time.Now()
	dd, err := controller.get(s.ID)
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
		// good
		default:
			t.Fatalf("Got an error other than NotFoundError when getting an expired service: %s\n", err)
		}
	} else {
		t.Fatalf("Service was not removed after 1 seconds. \nTTL: %v \nCreated: %v \nExpiry: %v \nNot deleted after: %v at %v\n",
			dd.TTL,
			dd.Created,
			dd.Expires,
			checkingTime.Sub(addingTime),
			checkingTime.UTC(),
		)
	}
}
