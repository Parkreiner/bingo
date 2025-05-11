import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App.tsx";

const root = document.getElementById("root");
if (root === null) {
  throw new Error("Root is not attached to DOM");
}

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>
);
