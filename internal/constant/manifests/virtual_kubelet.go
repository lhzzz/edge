package manifests

const VIRTUAL_KUBELET = "virtual_kubelet.yaml"

//证书是通过挂载的方式打进去的，这里需要修改
//因为virtual-kubelet需要和apiserver进行交互，创建Node
const VirtualKubeletYaml = `
---
apiVersion: v1
kind: Namespace
metadata:
  name: {{.NodeNamespace}}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.NodeName}}
  namespace: edge-cluster
  labels:
    k8s-app: {{.NodeName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      k8s-app: {{.NodeName}}
  template:
    metadata:
      labels:
        k8s-app: {{.NodeName}}
    spec:
      containers:
      - name: virtual-kubelet
        image: vk:latest
        command:
        - /home/virtual-kubelet
        args:
        - --nodename={{.NodeName}}
        - --namespace={{.NodeNamespace}}
        - --provider-config=cci.toml
        - --disable-taint=true
        - --provider=mock
        imagePullPolicy: IfNotPresent
        env:
          - name: APISERVER_CERT_LOCATION
            value: "/home/kubeapi_client.crt"
          - name: APISERVER_KEY_LOCATION
            value: "/home/kubeapi_client.key"
          - name: APISERVER_CA_CERT_LOCATION
            value: "/home/kubeapi_ca.crt"
        volumeMounts:
          - name: kube-config
            mountPath: /root/.kube/
            readOnly: true
      volumes:
      - name: kube-config
        hostPath:
          path: /root/.kube
          type: ""
      nodeSelector:
        virtual: "false"
---
kind: Service
apiVersion: v1
metadata:
  labels:
    k8s-app: {{.NodeName}}
  name: {{.NodeName}}
  namespace: edge-cluster
spec:
  ports:
  - name: kubelet
    port: 10251
    targetPort: 10251
  - name: http
  port: 80
  targetPort: 80
  selector:
    k8s-app: {{.NodeName}}
`
