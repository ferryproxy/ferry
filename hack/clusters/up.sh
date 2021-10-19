#!/usr/bin/env bash

dir="$(dirname "${BASH_SOURCE}")"
${dir}/kind/up.sh


out="${dir}/../../kubeconfig"
mkdir -p ${out}

ip="$(${dir}/host-docker-internal.sh)"
echo "Host: ${ip}"
for name in $(${dir}/kind/list.sh | grep -v control-); do
  kubectl --context="kind-${name}" config view --minify --raw=true > ${out}/"${name}"
  kubeconfig=$(cat ${out}/"${name}" | sed "s/127.0.0.1/${ip//[[:space:]]/}/g" | sed 's/certificate-authority-data: .\+/insecure-skip-tls-verify: true/g' | base64)
  # kubeconfig=$(cat ${out}/"${name}" | base64)
  sed -i "s/:.\+#<host-ip>/: ${ip//[[:space:]]/} #<host-ip>/g" ${dir}/control-plane-cluster/${name}.yaml
  sed -i "s/:.\+#<host-port:${name}>/: $(cat hack/clusters/kind/${name}.yaml | yq '.nodes[0].extraPortMappings[0].hostPort') #<host-port:${name}>/g" ${dir}/control-plane-cluster/${name}.yaml
  sed -i "s/:.\+#<base64-encoded-kubeconfig-data:${name}>/: ${kubeconfig} #<base64-encoded-kubeconfig-data:${name}>/g" ${dir}/control-plane-cluster/${name}.yaml
done

for name in $(${dir}/kind/list.sh); do
  kubectl --context "kind-${name}" apply -k "${dir}/${name}"
done


