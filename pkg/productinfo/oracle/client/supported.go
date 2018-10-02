// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import "fmt"

// GetSupportedShapes gives back supported node shapes in all subscribed regions for a service
// currently only 'compute' and 'oke' services are supported
func (oci *OCI) GetSupportedShapes(service string) (shapes map[string][]string, err error) {
	ic, err := oci.NewIdentityClient()
	if err != nil {
		return shapes, err
	}

	regions, err := ic.GetSubscribedRegionNames()
	if err != nil {
		return shapes, err
	}

	shapes = make(map[string][]string, 0)
	for _, region := range regions {
		_shapes, err := oci.GetSupportedShapesInARegion(region, service)
		if err != nil {
			return shapes, err
		}
		shapes[region] = _shapes
	}

	return shapes, err
}

// GetSupportedShapesInARegion gives back supported node shapes in the given region and service
// currently only 'compute' and 'oke' services are supported
func (oci *OCI) GetSupportedShapesInARegion(region, service string) (shapes []string, err error) {
	uniquemap := make(map[string]bool)

	err = oci.ChangeRegion(region)
	if err != nil {
		return shapes, err
	}

	_shapes := make([]string, 0)
	if service == "compute" {
		c, err := oci.NewComputeClient()
		if err != nil {
			return nil, err
		}
		pShapes, err := c.GetShapes()
		if err != nil {
			return nil, err
		}
		for _, pShape := range pShapes {
			_shapes = append(_shapes, *pShape.Shape)
		}
	} else if service == "oke" {
		ce, err := oci.NewContainerEngineClient()
		if err != nil {
			return nil, err
		}
		options, err := ce.GetDefaultNodePoolOptions()
		if err != nil {
			return nil, err
		}
		_shapes = options.Shapes.Get()
	}

	shapes = make([]string, 0)
	for _, shape := range _shapes {
		if _, ok := uniquemap[shape]; !ok {
			shapes = append(shapes, shape)
			uniquemap[shape] = true
		}
	}

	return shapes, nil
}

// GetSupportedImages gives back supported node images in all subscribed regions for a service
// currently only 'compute' and 'oke' services are supported
func (oci *OCI) GetSupportedImages(service string) (images map[string][]string, err error) {

	ic, err := oci.NewIdentityClient()
	if err != nil {
		return images, err
	}

	regions, err := ic.GetSubscribedRegionNames()
	if err != nil {
		return images, err
	}

	images = make(map[string][]string, 0)
	for _, region := range regions {
		_images, err := oci.GetSupportedImagesInARegion(region, service)
		if err != nil {
			return images, err
		}
		images[region] = _images
	}

	return images, err
}

// GetSupportedImagesInARegion gives back supported node images in the given region and service
// currently only 'compute' and 'oke' services are supported
func (oci *OCI) GetSupportedImagesInARegion(region, service string) (images []string, err error) {
	uniquemap := make(map[string]bool)

	err = oci.ChangeRegion(region)
	if err != nil {
		return nil, err
	}

	_images := make([]string, 0)
	if service == "compute" {
		c, err := oci.NewComputeClient()
		if err != nil {
			return nil, err
		}
		imgs, err := c.GetImages()
		if err != nil {
			return nil, err
		}
		for _, img := range imgs {
			_images = append(_images, fmt.Sprintf("%s %s", *img.OperatingSystem, *img.OperatingSystemVersion))
		}
	} else if service == "oke" {
		ce, err := oci.NewContainerEngineClient()
		if err != nil {
			return nil, err
		}

		options, err := ce.GetDefaultNodePoolOptions()
		if err != nil {
			return nil, err
		}

		_images = options.Images.Get()
	}

	for _, image := range _images {
		if _, ok := uniquemap[image]; !ok {
			images = append(images, image)
			uniquemap[image] = true
		}
	}

	return images, nil
}
