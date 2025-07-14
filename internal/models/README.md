# models package

The intent of this package is to have the base models that are used throughout all other packages. It's easy to end up with ugly import cycle issues when these models are sprinkled throughout other packages.

These models should remain simple structs and corresponding functions hanging off of those structs.
