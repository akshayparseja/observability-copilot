import React from "react";
import DashboardShell from "./components/DashboardShell";
import ImportForm from "./components/ImportForm";

function App() {
  return (
    <DashboardShell>
      <ImportForm />
      {/* Add other widgets/features here */}
    </DashboardShell>
  );
}
export default App;
