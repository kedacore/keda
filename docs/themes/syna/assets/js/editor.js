import React from 'react';
import ReactJsonschemaForm from 'react-jsonschema-form';

(window.synaPortals || (window.synaPortals = {})).editors = (window.editors || []).map(editor => {
  return {
    component: class Editor extends React.PureComponent {
      render() {
        return (
          <div className="container editor-container">
            <ReactJsonschemaForm schema={editor.schema} uiSchema={editor.ui} />
          </div>
        );
      }
    },
    container: editor.container,
  };
});
