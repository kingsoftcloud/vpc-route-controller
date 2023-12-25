FROM hub.kce.ksyun.com/tekton/alpine:3.6
ADD ./output/annotation /usr/bin/annotation
ENTRYPOINT ["/usr/bin/annotation"]