#!/usr/bin/env bash

dir="$(dirname "${BASH_SOURCE}")"
${dir}/kind/up.sh


out=$(realpath "${dir}/../../kubeconfig")
mkdir -p ${out}

conf=$(realpath "${dir}")
ip="$(${dir}/kind/host-docker-internal.sh)"

for name in $(${dir}/kind/list.sh | grep -v control-); do
  if [[ ! -f "${out}/${name}.yaml" ]] || [[ "$(cat "${out}/${name}.yaml" | grep "kind-${name}")" == "" ]]; then
    continue
  fi
  kubeconfig="$(cat "${out}/${name}" | base64)"
  sed -i "s/:.\+#<host-ip>/: ${ip//[[:space:]]/} #<host-ip>/g" ${dir}/control-plane-cluster/${name}.yaml
  sed -i "s/:.\+#<host-port:${name}>/: $(cat hack/clusters/kind/${name}.yaml | yq '.nodes[0].extraPortMappings[0].hostPort') #<host-port:${name}>/g" ${dir}/control-plane-cluster/${name}.yaml
  sed -i "s/:.\+#<base64-encoded-kubeconfig-data:${name}>/: ${kubeconfig} #<base64-encoded-kubeconfig-data:${name}>/g" ${dir}/control-plane-cluster/${name}.yaml
done

for name in $(ls ${out} | grep -v .yaml); do
  kubeconfig="${out}/${name}.yaml"
  if [[ ! -f "${out}/${name}.yaml" ]]; then
    kubeconfig="${out}/${name}"
  fi
  echo kubectl --kubeconfig "${kubeconfig}" apply -k "${conf}/${name}"
  kubectl --kubeconfig "${kubeconfig}" apply -k "${conf}/${name}"
done


