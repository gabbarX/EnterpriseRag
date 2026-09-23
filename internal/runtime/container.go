// Package runtime provides the dependency injection container for the
// application runtime. It uses uber's dig library to manage injection.
package runtime

import (
	"go.uber.org/dig"
)

// container is the application's global dependency injection container.
// Every service and component is registered and resolved through it.
var container *dig.Container

// init initialises the dependency injection container at process start-up.
func init() {
	container = dig.New()
}

// GetContainer returns a reference to the global dependency injection
// container, for other packages to register or fetch services.
func GetContainer() *dig.Container {
	return container
}
