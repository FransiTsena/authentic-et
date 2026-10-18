import React from 'react';
import ReactDOM from 'react-dom/client';
import { RouterProvider } from './router';
import { App } from './App';
import './index.css';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <RouterProvider>
      <App />
    </RouterProvider>
  </React.StrictMode>,
);

