FROM scratch

ADD dist/. /

ENTRYPOINT [ "/kore" ]