FROM busybox
COPY cmd /
ENTRYPOINT ["/cmd","--disableTLSVerification"]
