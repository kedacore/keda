+++
fragment = "content"
weight = 300

title = "Documentation"

[sidebar]
  sticky = true
+++

### Variables

#### header
*type: object*

This array specifies the header row (`thead > tr`) of the table.

##### header.values
*type: array of objects*

Each `header.values` object present in this array is to describe what the cell in the header row (`thead > tr > th`) looks like.

###### header.values.text
*type: string*

Title of the cell.

###### header.values.hide_on_mobile
*type: boolean*  
*default: false*

If set to `true`, the cell will not appear on smaller devices. The table behaves responsive to the width but using this feature you can lessen the width of the table, making it easier to navigate.

#### rows
*type: object*

This array specifies the rows after the header (`tbody > tr`) of the table.

##### rows.values
*type: array of objects*

Each `rows.values` object present in this array is to describe what the cell in the row (`tbody > tr > td`) looks like.

###### rows.values.text
*type: string*

Title of the cell. Will not appear if `rows.values.button` or `rows.values.icon` is set.

###### rows.values.icon
*type: string*

If set, the text value will be ignored and instead an icon will appear in the cell. This value will is the class name of that icon (checkout Fontawesome for more info on how to add icons).

###### rows.values.button
*type: string*

If set, the text or icon values will be ignored and instead a button will appear in the cell. This value will be shown as the title for that button.

###### rows.values.url
*type: string*

URL of the button if `rows.values.button` or `rows.values.icon` is set.

###### rows.values.align
*type: left, right, center*  
*default: center*

Specifies the horizontal alignment of the cell.

###### rows.values.hide_on_mobile
*type: boolean*  
*default: false*

If set to `true`, the cell will not appear on smaller devices. The table behaves responsive to the width but using this feature you can lessen the width of the table, making it easier to navigate.

[Global variables](/docs/global-variables) are documented as well and have been omitted from this page.
