import React, { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

function MyButton() {
  return (
    <button className='bg-green-600 px-4 py-2 rounded-full text-white' onClick={() => { alert('hi') }}>
      I'm a button
    </button>
  );
}

let App = function MyApp() {
  return (
    <div>
      <h1 className='text-sky-700'>React works</h1>
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
