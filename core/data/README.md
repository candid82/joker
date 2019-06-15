# Adding libraries

Besides dropping new `joker.X` library source code here, modify these to pick it up during the build:

* core/gen\_data/gen\_data.go `files` array
* core/procs.go `InitInternalLibs()` array

That will make it available at run time via e.g. `:require` in an `ns`, but not preloaded nor shown in `*loaded-libs*`.

Also add the coresponding `(require 'joker.x)` line to `generate-docs.joke`.

