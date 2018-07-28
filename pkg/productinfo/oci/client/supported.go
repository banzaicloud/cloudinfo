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

	err = oci.ChangeRegion(region)
	if err != nil {
		return shapes, err
	}

	ce, err := oci.NewContainerEngineClient()
	if err != nil {
		return nil, err
	}

	options, err := ce.GetDefaultNodePoolOptions()
	if err != nil {
		return nil, err
	}

	return options.Shapes.Get(), nil
}
