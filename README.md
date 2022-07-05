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

### Data plane pre-join

``` bash
# execute on control plane to pre-join other data plane
ferryctl control-plane pre-join direct <other-data-plane-name>
```
or
``` bash 
# execute on control plane to pre-join other data plane with tunnel mode
ferryctl control-plane pre-join tunnel <other-data-plane-name>
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
apiVersion: ferry.zsm.io/v1alpha1
kind: FerryPolicy
metadata:
  name: ferry-test
  namespace: ferry-system
spec:
  rules:
    - exports:
        - clusterName: cluster-1
          match:
            labels:
              app: web-1
      imports:
        - clusterName: control-plane
```

example2
``` yaml
# Mapping web-1.test.svc of cluster-1 to the xxx-1.default.svc of control-plane
apiVersion: ferry.zsm.io/v1alpha1
kind: FerryPolicy
metadata:
  name: ferry-test
  namespace: ferry-system
spec:
  rules:
    - exports:
        - clusterName: cluster-1
          match:
            namespace: test
            name: web-1
      imports:
        - clusterName: control-plane
          match:
            namespace: default
            name: xxx-1
```


