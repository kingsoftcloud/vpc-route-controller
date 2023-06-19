FROM hub.kce.ksyun.com/tekton/alpine:3.6
ADD ./output/vpc-route-controller /usr/bin/vpc-route-controller
ENTRYPOINT ["/usr/bin/vpc-route-controller"]

