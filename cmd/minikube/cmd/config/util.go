/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"fmt"
	"strconv"

	"github.com/docker/machine/libmachine"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/pkg/errors"
	"k8s.io/minikube/pkg/minikube/assets"
	"k8s.io/minikube/pkg/minikube/cluster"
	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/constants"
	"k8s.io/minikube/pkg/minikube/sshutil"
)

// Runs all the validation or callback functions and collects errors
func run(name string, value string, fns []setFn) error {
	var errors []error
	for _, fn := range fns {
		err := fn(name, value)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}

func findSetting(name string) (Setting, error) {
	for _, s := range settings {
		if name == s.name {
			return s, nil
		}
	}
	return Setting{}, fmt.Errorf("Property name %s not found", name)
}

// Set Functions

func SetString(m config.MinikubeConfig, name string, val string) error {
	m[name] = val
	return nil
}

func SetInt(m config.MinikubeConfig, name string, val string) error {
	i, err := strconv.Atoi(val)
	if err != nil {
		return err
	}
	m[name] = i
	return nil
}

func SetBool(m config.MinikubeConfig, name string, val string) error {
	b, err := strconv.ParseBool(val)
	if err != nil {
		return err
	}
	m[name] = b
	return nil
}

func EnableOrDisableAddon(name string, val string) error {
	enable, err := strconv.ParseBool(val)
	if err != nil {
		errors.Wrapf(err, "error attempted to parse enabled/disable value addon %s", name)
	}
	api := libmachine.NewClient(constants.Minipath, constants.MakeMiniPath("certs"))
	defer api.Close()
	cluster.EnsureMinikubeRunningOrExit(api, 0)

	addon, _ := assets.Addons[name] // validation done prior
	if err != nil {
		return err
	}
	host, err := cluster.CheckIfApiExistsAndLoad(api)
	if enable {
		if err = transferAddonViaDriver(addon, host.Driver); err != nil {
			return errors.Wrapf(err, "Error transferring addon %s to VM", name)
		}
	} else {
		if err = deleteAddonViaDriver(addon, host.Driver); err != nil {
			return errors.Wrapf(err, "Error deleting addon %s from VM", name)
		}
	}
	return nil
}

func deleteAddonViaDriver(addon *assets.Addon, d drivers.Driver) error {
	client, err := sshutil.NewSSHClient(d)
	if err != nil {
		return err
	}
	if err := sshutil.DeleteAddon(addon, client); err != nil {
		return err
	}
	return nil
}

func transferAddonViaDriver(addon *assets.Addon, d drivers.Driver) error {
	client, err := sshutil.NewSSHClient(d)
	if err != nil {
		return err
	}
	if err := sshutil.TransferAddon(addon, client); err != nil {
		return err
	}
	return nil
}
