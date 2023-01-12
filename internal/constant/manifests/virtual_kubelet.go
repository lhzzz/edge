package manifests

const VIRTUAL_KUBELET = "virtual_kubelet.yaml"

//因为virtual-kubelet需要和apiserver进行交互，创建Node
const VirtualKubeletYaml = `
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vk-{{.NodeName}}
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
        image: registry.edge.com/cloud-native/virtual-kubelet:latest
        command:
        - /home/virtual-kubelet
        args:
        - --nodename={{.NodeName}}
        - --provider-config=/home/vk-config/cci.toml
        - --provider=edge
        imagePullPolicy: Always
        env:
        - name: CLUSTER_POD_IP
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: status.podIP
        volumeMounts:
          - name: kube-config
            mountPath: /root/.kube/
            readOnly: true
          - name: cci
            mountPath: /home/vk-config
          - name: localtime
            mountPath: /etc/localtime    
      dnsPolicy: None
      dnsConfig:
        nameservers:
        - 10.96.0.12
      volumes:
      - name: kube-config
        hostPath:
          path: /root/.kube
          type: ""
      - name: cci
        configMap:
          name: vk-config
      - name: localtime
        hostPath:
          path: /etc/localtime
          type: ""
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
  selector:
    k8s-app: vk-{{.NodeName}}
`
