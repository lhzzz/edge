package manifests

const EDGE_INGRESS = "edge_ingress.yaml"

const EdgeIngressYaml = `
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-body-size: "48m"
    nginx.ingress.kubernetes.io/cors-allow-headers: >-
        origin,range,accept-encoding,referer,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,X-Forwarded-For,x-grpc-web,x-user-agent,If-Modified-Since,Cache-Control,Content-Type,Authorization
    nginx.ingress.kubernetes.io/cors-allow-methods: 'PUT, GET, POST, OPTIONS, HEAD, DELETE,'
    nginx.ingress.kubernetes.io/cors-allow-credentials: 'true'
    nginx.ingress.kubernetes.io/cors-allow-origin: '*'
    nginx.ingress.kubernetes.io/enable-cors: 'true'
  name: edge-ingress
spec:
  ingressClassName: nginx
  rules:
  - http:
      paths:
      - path: /edge/registry
        pathType: Prefix
        backend:
          service:
            name: edge-registry
            port:
              number: 80
`
