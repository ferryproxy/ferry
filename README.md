# Ferry

Ferry is a multi-cluster communication component of Kubernetes that supports mapping services from one cluster to another.

## Quick Start

### Download ferryctl (ferry install management tool)

https://github.com/ferry-proxy/ferry/releases

## Initialize control plane

``` bash
# execute on control plane
ferryctl control-plane init
```

## Other data plane join

### Data plane join

``` bash
# execute on control plane to join other data plane
ferryctl control-plane join <other-data-plane-name>
```

PS: The `<other-data-plane-name>` is the same as FerryPoliy used below


### Data plane join

After the last command is executed of the **Control Plane**, it responds with a command, copied to the **Data Plane** to run.

### Control plane to control the data plane

After the last command is executed of the **Data Plane**, it responds with a command, copied to the the **Control Plane** to run.

### Configuration rules

All rules in control plane

example1
``` yaml
# Mapping services of match label app=web-1 of cluster-1 to the control-plane
apiVersion: traffic.ferry.zsm.io/v1alpha2
kind: RoutePolicy
metadata:
  name: ferry-test
  namespace: ferry-system
spec:
  exports:
    - hubName: cluster-1
      service:
        labels:
          app: web-1
  imports:
    - hubName: control-plane
```

example2
``` yaml
# Mapping web-1.test.svc of cluster-1 to the xxx-1.default.svc of control-plane
apiVersion: traffic.ferry.zsm.io/v1alpha2
kind: RoutePolicy
metadata:
  name: ferry-test
  namespace: ferry-system
spec:
  exports:
    - hubName: cluster-1
      service:
        namespace: test
        name: web-1
  imports:
    - hubName: control-plane
      service:
        namespace: default
        name: xxx-1
```


