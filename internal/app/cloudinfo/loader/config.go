package loader

type Config struct {
	// the locations - folder on the filesystem where the loader looks for data to be loaded
	Location string

	// the name of the file with the data
	Name string

	// the format of the data file (json / yaml)
	Format string
}
