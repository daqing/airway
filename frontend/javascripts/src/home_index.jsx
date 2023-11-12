import React, { StrictMode, useState } from 'react';
import { createRoot } from 'react-dom/client';

function MyButton({ setMsg }) {
  return (
    <button className='bg-green-600 px-4 py-2 rounded-full text-white' onClick={() => { setMsg('Hello, React!') }}>
      React works
    </button>
  );
}

let App = function MyApp() {
  const [msg, setMsg] = useState("");

  return (
    <div className='mt-10 text-center'>
      <MyButton setMsg={setMsg} />
      <div className="mt-5 text-green-700 font-mono">{msg}</div>
    </div>
  );
}


const root = createRoot(document.getElementById('react-root'));
root.render(
  <StrictMode>
    <App />
  </StrictMode>
);
