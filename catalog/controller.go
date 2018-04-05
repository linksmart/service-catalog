// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"code.linksmart.eu/sc/service-catalog/utils"
	avl "github.com/ancientlore/go-avltree"
	uuid "github.com/satori/go.uuid"
)

type Controller struct {
	wg sync.WaitGroup
	sync.RWMutex
	storage   Storage
	listeners []Listener
	ticker    *time.Ticker

	// sorted expiryTime->serviceID maps
	exp_sid *avl.Tree
}

func NewController(storage Storage, listeners ...Listener) (*Controller, error) {
	c := Controller{
		storage:   storage,
		exp_sid:   avl.New(timeKeys, avl.AllowDuplicates), // allows more than one service with the same expiry time
		listeners: listeners,
	}

	// Initialize secondary indices (if a persistent storage backend is present)
	err := c.initIndices()
	if err != nil {
		return nil, err
	}

	c.ticker = time.NewTicker(5 * time.Second)
	go c.cleanExpired()

	return &c, nil
}

func (c *Controller) add(s Service) (*Service, error) {
	if err := s.validate(); err != nil {
		return nil, &BadRequestError{err.Error()}
	}

	c.Lock()
	defer c.Unlock()

	if s.ID == "" {
		// System generated id
		s.ID = uuid.NewV4().String()
	}
	s.Created = time.Now().UTC()
	s.Updated = s.Created
	if s.TTL == 0 {
		s.Expires = nil
	} else {
		expires := s.Created.Add(time.Duration(s.TTL) * time.Second)
		s.Expires = &expires
	}

	err := c.storage.add(&s)
	if err != nil {
		return nil, err
	}

	// Add secondary indices
	c.addIndices(&s)

	// notify listeners
	for _, l := range c.listeners {
		go l.added(s)
	}

	return &s, nil
}

func (c *Controller) get(id string) (*Service, error) {
	return c.storage.get(id)
}

func (c *Controller) update(id string, s Service) (*Service, error) {
	if err := s.validate(); err != nil {
		return nil, &BadRequestError{err.Error()}
	}

	c.Lock()
	defer c.Unlock()

	// Get the stored service
	ss, err := c.storage.get(id)
	if err != nil {
		return nil, err
	}

	// Shallow copy
	var cp Service = *ss

	ss.Description = s.Description
	ss.Name = s.Name
	ss.APIs = s.APIs
	ss.Docs = s.Docs
	ss.Meta = s.Meta
	ss.TTL = s.TTL
	ss.Updated = time.Now().UTC()
	if ss.TTL == 0 {
		ss.Expires = nil
	} else {
		expires := ss.Updated.Add(time.Duration(ss.TTL) * time.Second)
		ss.Expires = &expires
	}

	err = c.storage.update(id, ss)
	if err != nil {
		return nil, err
	}

	// Update secondary indices
	c.removeIndices(&cp)
	c.addIndices(ss)

	// notify listeners
	for _, l := range c.listeners {
		go l.updated(s)
	}

	return ss, nil
}

func (c *Controller) delete(id string) error {
	c.Lock()
	defer c.Unlock()

	old, err := c.storage.get(id)
	if err != nil {
		return err
	}

	err = c.storage.delete(id)
	if err != nil {
		return err
	}

	// Remove secondary indices
	c.removeIndices(old)

	// notify listeners
	for _, l := range c.listeners {
		go l.deleted(*old)
	}

	return nil
}

func (c *Controller) list(page, perPage int) ([]Service, int, error) {
	return c.storage.list(page, perPage)
}

func (c *Controller) filter(path, op, value string, page, perPage int) ([]Service, int, error) {
	c.RLock()
	defer c.RUnlock()

	matches := make([]Service, 0)
	pp := MaxPerPage
	for p := 1; ; p++ {
		services, t, err := c.storage.list(p, pp)
		if err != nil {
			return nil, 0, err
		}

		for i := range services {
			matched, err := utils.MatchObject(services[i], strings.Split(path, "."), op, value)
			if err != nil {
				return nil, 0, err
			}
			if matched {
				matches = append(matches, services[i])
			}
		}

		if p*pp >= t {
			break
		}
	}
	// Pagination
	offset, limit, err := utils.GetPagingAttr(len(matches), page, perPage, MaxPerPage)
	if err != nil {
		return nil, 0, &BadRequestError{fmt.Sprintf("Unable to paginate: %s", err)}
	}
	// Return the page
	return matches[offset : offset+limit], len(matches), nil
}

func (c *Controller) total() (int, error) {
	return c.storage.total()
}

func (c *Controller) cleanExpired() {
	for t := range c.ticker.C {
		c.Lock()

		var expiredList []Map
		for m := range c.exp_sid.Iter() {
			if !m.(Map).key.(time.Time).After(t.UTC()) {
				expiredList = append(expiredList, m.(Map))
			} else {
				// exp_did is sorted by time ascendingly: its elements expire in order
				break
			}
		}

		for _, m := range expiredList {
			id := m.value.(string)
			logger.Printf("cleanExpired() Registration %v has expired\n", id)

			old, err := c.storage.get(id)
			if err != nil {
				logger.Printf("cleanExpired() Error retrieving device %v: %v\n", id, err.Error())
				break
			}

			err = c.storage.delete(id)
			if err != nil {
				logger.Printf("cleanExpired() Error removing device %v: %v\n", id, err.Error())
				break
			}
			// Remove secondary indices
			c.removeIndices(old)

			// notify listeners
			for _, l := range c.listeners {
				go l.deleted(*old)
			}
		}

		c.Unlock()
	}
}

func (c *Controller) AddListener(listener Listener){
	c.Lock()
	c.listeners = append(c.listeners, listener)
	c.Unlock()
}

func (c *Controller) RemoveListener(listener Listener){
	c.Lock()
	for i, l := range c.listeners {
		if(l == listener ){
			//delete the entry and break
			c.listeners = append(c.listeners[:i], c.listeners[i+1:]...)
			break
		}
	}
	c.Unlock()
}
// Stop the controller
func (c *Controller) Stop() error {
	c.ticker.Stop()
	return c.storage.Close()
}

// UTILITY FUNCTIONS

// Initialize secondary indices (from a persistent storage backend)
func (c *Controller) initIndices() error {
	perPage := MaxPerPage
	for page := 1; ; page++ {
		devices, total, err := c.storage.list(page, perPage)
		if err != nil {
			return err
		}

		for i, _ := range devices {
			c.addIndices(&devices[i])
		}

		if page*perPage >= total {
			break
		}
	}
	return nil
}

// Creates secondary indices
// WARNING: the caller must obtain the lock before calling
func (c *Controller) addIndices(s *Service) {

	// Add expiry time index
	if s.TTL != 0 {
		c.exp_sid.Add(Map{*s.Expires, s.ID})
	}
}

// Removes secondary indices
// WARNING: the caller must obtain the lock before calling
func (c *Controller) removeIndices(s *Service) {

	// Remove the expiry time index
	// INFO:
	// More than one service can have the same expiry time (i.e. map's key)
	//	which leads to non-unique keys in the maps.
	// This code removes keys with that expiry time (keeping them in a temp) until the
	// 	desired target is reached. It then adds the items in the temp back to the tree.
	if s.TTL != 0 {
		var temp []Map
		for m := range c.exp_sid.Iter() {
			id := m.(Map).value.(string)
			if id == s.ID {
				for { // go through all duplicates (same expiry times)
					r := c.exp_sid.Remove(m)
					if r == nil {
						break
					}
					if id != s.ID {
						temp = append(temp, r.(Map))
					}
				}
				break
			}
		}
		for _, r := range temp {
			c.exp_sid.Add(r)
		}
	}
}

// AVL Tree: sorted nodes according to keys
//
// A node of the AVL Tree (go-avltree)
type Map struct {
	key   interface{}
	value interface{}
}

// Operator for Time-type key
func timeKeys(a interface{}, b interface{}) int {
	if a.(Map).key.(time.Time).Before(b.(Map).key.(time.Time)) {
		return -1
	} else if a.(Map).key.(time.Time).After(b.(Map).key.(time.Time)) {
		return 1
	}
	return 0
}
