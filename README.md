Kooper
======

Kooper is a simple Go library to create Kubernetes [operators](https://coreos.com/operators/).

## Features.

* Easy and decoupled library.
* Remove all duplicated code from every controller/operator.
* Uses the tooling already created by Kubernetes.
* Remove complexity from operators and focus on domain logic.
* Compatible to create [controllers](https://github.com/kubernetes/community/blob/master/contributors/devel/controllers.md) also.
* Don't support TPRs (yes it's afeature)


## Motivation.

The state of art in the operators/controllers moves fast, a lot of new operators are being published every day. Most of them have the same "infrastructure" code refering Kubernetes operators/controllers and bootstrapping a new operator can be slow or repetitive.

At this moment there isn't an standard although there are some projects like [rook operator kit](https://github.com/rook/operator-kit) or [Giantswarm operator kit](https://github.com/giantswarm/operatorkit) that are trying to create it.

At Spotahome we studied these projects before developing Kooper and they didn't follow the way we wanted to decouple our operators and abstract most of the Kubernetes operators/controllers code so we can focus on the domain logic, and we want to reduce the complexity in every part of the stack including code.