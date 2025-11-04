import React from "react";
import Sidebar from "./Sidebar";
import Topbar from "./Topbar";
import { Box, Toolbar } from "@mui/material";

export default function DashboardShell({ children }: { children: React.ReactNode }) {
  return (
    <Box sx={{ display: "flex" }}>
      <Topbar />
      <Sidebar />
      <Box component="main" sx={{ flexGrow: 1, p: 5, bgcolor: "#f8fafc", minHeight: "100vh" }}>
        <Toolbar />
        {children}
      </Box>
    </Box>
  );
}
