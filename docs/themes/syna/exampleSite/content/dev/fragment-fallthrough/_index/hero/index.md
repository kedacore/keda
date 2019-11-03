+++
fragment = "hero"
#disabled = true
date = "2016-09-07"
weight = 50
background = "light" # can influence the text color
particles = true

title = "Rendered from a directory in page"
subtitle = "If you see this, directory fragment fallthrough is working"

[header]
  image = "header.jpg"

[asset]
  image = "logo.svg"
  width = "500px" # optional - will default to image width
  #height = "150px" # optional - will default to image height

[[buttons]]
  text = "Features"
  url = "#features"
  color = "dark" # primary, secondary, success, danger, warning, info, light, dark, link - default: primary

[[buttons]]
  text = "Getting Started"
  url = "/docs/"
  color = "primary"

[[buttons]]
  text = "Fragments"
  url = "#fragments"
  color = "dark"
+++
