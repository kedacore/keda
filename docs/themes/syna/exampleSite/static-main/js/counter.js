import * as React from 'react';

class Counter extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      counter: 0,
    };
    this.increment = this.increment.bind(this);
    this.decrement = this.decrement.bind(this);
  }

  increment() {
    this.setState({ counter: this.state.counter + 1 });
  }

  decrement() {
    this.setState({ counter: this.state.counter - 1 });
  }

  render() {
    return (
      <div className="container py-4 text-center">
        <h3 className="text-dark">React Portal fragment filled with a custom React component</h3>
        <h4 className="text-dark">This fragment supports React components and renders them in a portal.</h4>
        <div className="row justify-content-center mt-4">
          <div className="col">
            <p className="text-center">Counter: {' '}<b>{this.state.counter}</b></p>
            <div className="row justify-content-center">
              <button className="decrement btn btn-primary mr-2" onClick={this.decrement}>Decrement -</button>
              <button className="increment btn btn-primary ml-2" onClick={this.increment}>+ Increment</button>
            </div>
          </div>
        </div>
      </div>
    );
  }
}

(window.synaPortals || (window.synaPortals = {})).counter = {
  component: Counter,
  container: '#counter [data-portal]',
};
