k8s-export
==========

k8s-export is a tool that exports YAML from an etcd database snapshot.
It can be useful if you need to restore specific things from an etcd
backup, rather than necessarily restoring everything.

Note that I haven't tried to cover every single thing that you might
have in your snapshot.  If k8s-export finds things it doesn't know
about, it will tell you by outputting e.g.

  Unknown some-new-api.k8s.ip/v1alpha3/SomeThingOrOther

If you want it to extract YAML for that thing, you will need to add
the necessary code to the writeYAML() function.

Ideally at some point it would be nice to write a code generator and
have it scan the k8s sources to find everything, but for now I don't
need that, and actually *you* probably don't need that either.

To use the tool, just run

  ./k8s-export -db <path-to-db-file> -o <name-of-output-folder>

k8s-export will create the output folder if it doesn't exist, and will
proceed to extract data from the database into it.

k8s-export extracts *ALL VERSIONS* of objects that it finds; the
version numbers will be put into the filenames, e.g.

  <output-folder>/registry/thing/namespace/name-of-thing-1234-0.yaml

has main version 1234 and sub version 0.
