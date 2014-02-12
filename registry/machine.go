package registry

import (
	"path"
	"time"

	"github.com/coreos/fleet/third_party/github.com/coreos/go-etcd/etcd"

	"github.com/coreos/fleet/event"
	"github.com/coreos/fleet/machine"
)

const (
	machinePrefix = "/machines/"
)

// Describe all active Machines
func (r *Registry) GetActiveMachines() []machine.Machine {
	key := path.Join(keyPrefix, machinePrefix)
	resp, err := r.etcd.Get(key, false, true)

	var machines []machine.Machine

	// Assume the error was KeyNotFound and return an empty data structure
	if err != nil {
		return machines
	}

	for _, kv := range resp.Node.Nodes {
		_, bootId := path.Split(kv.Key)
		mach := r.GetMachineState(bootId)
		if mach != nil {
			machines = append(machines, *mach)
		}
	}

	return machines
}

// Get Machine object from etcd
func (r *Registry) GetMachineState(bootid string) *machine.Machine {
	key := path.Join(keyPrefix, machinePrefix, bootid, "object")
	resp, err := r.etcd.Get(key, false, true)

	// Assume the error was KeyNotFound and return an empty data structure
	if err != nil {
		return nil
	}

	var mach machine.Machine
	if err := unmarshal(resp.Node.Value, &mach); err != nil {
		return nil
	}

	return &mach
}

// Push Machine object to etcd
func (r *Registry) SetMachineState(machine *machine.Machine, ttl time.Duration) {
	//TODO: Handle the error generated by marshal
	json, _ := marshal(machine)
	key := path.Join(keyPrefix, machinePrefix, machine.BootId, "object")

	// Assume state is already present, returning on success
	_, err := r.etcd.Update(key, json, uint64(ttl.Seconds()))
	if err == nil {
		return
	}

	// If state was not present, explicitly create it so the other members
	// in the cluster know this is a new member
	r.etcd.Create(key, json, uint64(ttl.Seconds()))
}

// Remove Machine object from etcd
func (r *Registry) RemoveMachineState(machine *machine.Machine) error {
	key := path.Join(keyPrefix, machinePrefix, machine.BootId, "object")
	_, err := r.etcd.Delete(key, false)
	return err
}

func (self *EventStream) filterEventMachineCreated(resp *etcd.Response) *event.Event {
	if base := path.Base(resp.Node.Key); base != "object" {
		return nil
	}

	if resp.Action != "create" {
		return nil
	}

	var m machine.Machine
	unmarshal(resp.Node.Value, &m)
	return &event.Event{"EventMachineCreated", m, nil}
}

func (self *EventStream) filterEventMachineRemoved(resp *etcd.Response) *event.Event {
	if base := path.Base(resp.Node.Key); base != "object" {
		return nil
	}

	if resp.Action != "expire" && resp.Action != "delete" {
		return nil
	}

	machName := path.Base(path.Dir(resp.Node.Key))
	return &event.Event{"EventMachineRemoved", machName, nil}
}
