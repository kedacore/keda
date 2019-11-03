+++
fragment = "content"
#disabled = true
date = "2016-09-07"
weight = 141
background = "light"

title = "Content with sidebar"
subtitle = "Split in two!"
#title_align = "left" # Default is center, can be left, right or center

[sidebar]
  title = "Sidebar"
  align = "left"
  #sticky = true # Default is false
  content = """
So much information  
Phone numbers  
Details  
Or even more  
Event with [a link](#)
"""
+++

Lorem ipsum dolor sit amet, [consectetur](#) adipiscing elit. *Curabitur a lorem urna.* **Quisque in neque malesuada**, sollicitudin nunc porttitor, ornare est. Praesent ante enim, bibendum sed hendrerit et, iaculis laoreet felis. `#title_align = "left" # Default is center, can be left, right or center`, Morbi efficitur dui sit amet orci porttitor, nec tincidunt turpis elementum. Suspendisse rutrum, mi ac sollicitudin blandit, eros sem tincidunt enim, vitae feugiat turpis eros ut diam. Nunc hendrerit, nibh vitae dignissim pretium, magna nulla lacinia massa, et interdum lacus purus ultricies lacus. Nulla tincidunt quis lacus in posuere. Integer urna lorem, ultricies ut est vel, rhoncus euismod metus. Vestibulum luctus maximus massa, ut egestas est iaculis in. Nunc nisi dolor, sodales et imperdiet ut, lacinia ac justo. Phasellus ultrices risus cursus maximus lobortis. Vestibulum sagittis elementum dignissim. Suspendisse iaculis `background = "secondary"` venenatis nisl, sed bibendum urna. Aliquam quis pellentesque tortor. Sed sed cursus nisl. Aenean eu lorem condimentum, feugiat mauris vitae, hendrerit tellus.

```
+++
fragment = "content"
#disabled = true
date = "2017-10-05"
weight = 110
background = "secondary"

title = "Content without sidebar"
subtitle = "Full width content fragment"
#title_align = "left" # Default is center, can be left, right or center
+++
```
