import React from "react";
import { AppBar, Toolbar, Typography, Avatar, Box } from "@mui/material";

export default function Topbar() {
  return (
    <AppBar position="fixed" color="inherit" elevation={2}>
      <Toolbar>
        <Typography variant="h6" component="div" sx={{ flexGrow: 1, fontWeight: 800 }}>
          Observability
        </Typography>
        <Box>
          <Avatar alt="User" src="https://i.pravatar.cc/40?img=7" />
        </Box>
      </Toolbar>
    </AppBar>
  );
}
