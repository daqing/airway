import React, { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

function MyButton() {
  return (
    <button>
      I'm a button
    </button>
  );
}

let App = function MyApp() {
  return (
    <div>
      <h1>Welcome to my app</h1>
      <MyButton />
    </div>
  );
}


const root = createRoot(document.getElementById('react-root'));
root.render(
  <StrictMode>
    <App />
  </StrictMode>
);
