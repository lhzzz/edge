package manifests

const VIRTUAL_KUBELET = "virtual_kubelet.yaml"

//因为virtual-kubelet需要和apiserver进行交互，创建Node
const VirtualKubeletYaml = `
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.NodeName}}
  namespace: edge-cluster
  labels:
    k8s-app: vk-{{.NodeName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      k8s-app: vk-{{.NodeName}}
  template:
    metadata:
      labels:
        k8s-app: vk-{{.NodeName}}
    spec:
      containers:
      - name: virtual-kubelet
        image: registry.sakura.com/cloud-native/virtual-kubelet:latest
        command:
        - /home/virtual-kubelet
        args:
        - --nodename={{.NodeName}}
        - --provider-config=/home/vk-config/cci.toml
        - --provider=zhst
        imagePullPolicy: IfNotPresent
        volumeMounts:
          - name: kube-config
            mountPath: /root/.kube/
            readOnly: true
          - name: cci
            mountPath: /home/vk-config
      volumes:
      - name: kube-config
        hostPath:
          path: /root/.kube
          type: ""
      - name: cci
        configMap:
          name: vk-config
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: type
                operator: NotIn
                values:
                - virtual-kubelet
---
kind: Service
apiVersion: v1
metadata:
  labels:
    k8s-app: vk-{{.NodeName}}
  name: vk-{{.NodeName}}
  namespace: edge-cluster
spec:
  ports:
  - name: kubelet
    port: 10250
    targetPort: 10250
#  - name: http
#    port: 80
#    targetPort: 80
  selector:
    k8s-app: vk-{{.NodeName}}
`
