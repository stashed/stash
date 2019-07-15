# Best practices

## Configuring Stow once

It is recommended that you create a single file that imports Stow and all the implementations to save you from doing so in every code file where you might use Stow. You can also take the opportunity to abstract the top level Stow methods that your code uses to save other code files (including other packages) from importing Stow at all.

Create a file called `storage.go` in your package and add the following code:

```go
import (
	"gomodules.xyz/stow"
	// support Azure storage
	_ "gomodules.xyz/stow/azure"
	// support Google storage
	_ "gomodules.xyz/stow/google"
	// support local storage
	_ "gomodules.xyz/stow/local"
	// support swift storage
	_ "gomodules.xyz/stow/swift"
	// support s3 storage
	_ "gomodules.xyz/stow/s3"
	// support oracle storage
	_ "gomodules.xyz/stow/oracle"
)

// Dial dials stow storage.
// See stow.Dial for more information.
func Dial(kind string, config stow.Config) (stow.Location, error) {
	return stow.Dial(kind, config)
}
```
