import React from 'react';
import ReactDOM from 'react-dom';
import './editor';

const Portal = ({ component: Component, container }) => {
  return ReactDOM.createPortal(
    <div>
      <Component />
    </div>
  , document.querySelector(container));
};

class Portals extends React.PureComponent {
  render() {
    return Object.keys(window.synaPortals || {}).map(portal => {
      if (Array.isArray(window.synaPortals[portal]) && window.synaPortals[portal].length > 0) {
        return window.synaPortals[portal].map(innerPortal => {
          if (document.querySelector(innerPortal.container) === null) {
            return null;
          }
  
          return (
            <Portal
              key={innerPortal.container}
              component={innerPortal.component}
              container={innerPortal.container} />
          );
        })
      }

      if (document.querySelector(window.synaPortals[portal].container) === null) {
        return null;
      }

      return (
        <Portal
          key={window.synaPortals[portal].container}
          component={window.synaPortals[portal].component}
          container={window.synaPortals[portal].container} />
      );
    });
  }
}

ReactDOM.render(<Portals />, document.querySelector('#react'));
