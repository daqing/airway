import React, { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

let App = () => {
  return (
    <div>
      <h1>Hello, sign_in - Index</h1>
    </div>
  );
}

const root = createRoot(document.getElementById('react-root'));

root.render(
  <StrictMode>
    <App />
  </StrictMode>
);
