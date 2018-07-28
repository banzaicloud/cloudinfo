package client

// GetSupportedShapes gives back supported node shapes in all subscribed regions
func (oci *OCI) GetSupportedShapes() (shapes map[string][]string, err error) {

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
		_shapes, err := oci.GetSupportedShapesInARegion(region)
		if err != nil {
			return shapes, err
		}
		shapes[region] = _shapes
	}

	return shapes, err
}

// GetSupportedShapesInARegion gives back supported node shapes in the given region
func (oci *OCI) GetSupportedShapesInARegion(region string) (shapes []string, err error) {

	uniquemap := make(map[string]bool)

	err = oci.ChangeRegion(region)
	if err != nil {
		return shapes, err
	}

	c, err := oci.NewComputeClient()
	if err != nil {
		return nil, err
	}

	_shapes, err := c.GetShapes()
	if err != nil {
		return nil, err
	}

	shapes = make([]string, 0)
	for _, shape := range _shapes {
		if _, ok := uniquemap[*shape.Shape]; !ok {
			shapes = append(shapes, *shape.Shape)
			uniquemap[*shape.Shape] = true
		}
	}

	return shapes, nil
}
