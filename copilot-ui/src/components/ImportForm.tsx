import React, { useState, useEffect } from "react";
import {
  Box,
  Button,
  Card,
  CardContent,
  Typography,
  TextField,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Paper,
  Radio,
  RadioGroup,
  FormControlLabel,
  FormControl,
  FormLabel,
  InputAdornment,
  CircularProgress,
  Chip,
  Alert,
} from "@mui/material";
import FolderIcon from "@mui/icons-material/Folder";
import SearchIcon from "@mui/icons-material/Search";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import CancelIcon from "@mui/icons-material/Cancel";
import axios from "axios";
import API_ENDPOINTS from "../config/api";

interface Repo {
  id: string;
  name: string;
}

interface ScanResult {
  framework: string;
  has_metrics: boolean;
  has_otel: boolean;
  services: string[];
}

interface ServiceInfo {
  name: string;
  framework: string;
  has_metrics: boolean;
  has_otel: boolean;
}

const telemetryModes = [
  {
    value: "metrics",
    label: "Metrics (Prometheus)",
    description: "Collect and visualize metrics only",
  },
  {
    value: "traces",
    label: "Traces (OpenTelemetry)",
    description: "Enable distributed tracing",
  },
  {
    value: "both",
    label: "Both (Hybrid)",
    description: "Combine Prometheus metrics with OTel traces/logs",
  },
  {
    value: "none",
    label: "None",
    description: "Disable telemetry collection",
  },
];

export default function ImportForm({
  onImportComplete,
}: {
  onImportComplete?: (data?: any) => void;
}) {
  const [repos, setRepos] = useState<Repo[]>([]);
  const [search, setSearch] = useState("");
  const [githubURL, setGithubURL] = useState("");
  const [selectedMode, setSelectedMode] = useState("none");
  const [loading, setLoading] = useState(false);
  const [fetchingRepos, setFetchingRepos] = useState(true);

  const [selectedRepo, setSelectedRepo] = useState<Repo | null>(null);
  const [updatingMode, setUpdatingMode] = useState(false);
  const [serviceInfo, setServiceInfo] = useState<ServiceInfo | null>(null);

  // NEW: Scan result state
  const [scanResult, setScanResult] = useState<ScanResult | null>(null);
  const [scanning, setScanning] = useState(false);
  const [showScanResult, setShowScanResult] = useState(false);

  useEffect(() => {
    fetchRepos();
  }, []);

  const fetchRepos = async () => {
    setFetchingRepos(true);
    try {
      const response = await axios.get(API_ENDPOINTS.REPOS);
      setRepos(response.data || []);
    } catch (error) {
      console.error("Failed to fetch repos:", error);
      setRepos([]);
    } finally {
      setFetchingRepos(false);
    }
  };

  const filteredRepos = repos.filter((repo) =>
    repo.name.toLowerCase().includes(search.toLowerCase())
  );

  // NEW: Scan repo first before import
  const handleScan = async () => {
    if (!githubURL) return;

    setScanning(true);
    setShowScanResult(false);
    setScanResult(null);

    try {
      // Call a new endpoint that just scans without importing
      // For now, we'll use the import endpoint to get scan result
      const response = await axios.post(API_ENDPOINTS.IMPORTS, {
        github_url: githubURL,
        telemetry_mode: "none", // Temporary - we'll update after user selects
      });

      const result = response.data.result as ScanResult;
      setScanResult(result);
      setShowScanResult(true);

      // Auto-select default mode based on scan result
      const defaultMode = getDefaultTelemetryMode(result);
      setSelectedMode(defaultMode);

    } catch (error) {
      console.error("Scan failed:", error);
      alert("Scan failed. Check console for details.");
    } finally {
      setScanning(false);
    }
  };

  // Determine default telemetry mode based on scan
  const getDefaultTelemetryMode = (result: ScanResult): string => {
    const { has_metrics, has_otel } = result;

    if (has_metrics && has_otel) return "both";
    if (has_metrics) return "metrics";
    if (has_otel) return "traces";
    return "none";
  };

  // Import with selected mode
  const handleImport = async () => {
    if (!githubURL || !scanResult) return;

    setLoading(true);
    try {
      await axios.post(API_ENDPOINTS.IMPORTS, {
        github_url: githubURL,
        telemetry_mode: selectedMode,
      });

      console.log("Import successful with mode:", selectedMode);
      
      // Reset form
      setGithubURL("");
      setSelectedMode("none");
      setScanResult(null);
      setShowScanResult(false);
      
      await fetchRepos();
      if (onImportComplete) onImportComplete();
    } catch (error) {
      console.error("Import failed:", error);
      alert("Import failed. Check console for details.");
    } finally {
      setLoading(false);
    }
  };

  // On clicking repo in list for editing
  const onSelectRepo = async (repo: Repo) => {
    setSelectedRepo(repo);
    setServiceInfo(null);

    try {
      // Fetch plan to get service info
      const planRes = await axios.get(API_ENDPOINTS.PLAN(repo.id));
      const services = planRes.data.services || [];

      if (services.length > 0) {
        setServiceInfo(services[0]); // Use first service
      }

      // Fetch current toggle mode
      const svcName = services[0]?.name || repo.name;
      const toggleRes = await axios.get(
        API_ENDPOINTS.TOGGLES(repo.id, svcName, "dev")
      );
      setSelectedMode(toggleRes.data.telemetry_mode || "none");
    } catch (error) {
      console.error("Failed to fetch repo details:", error);
      setSelectedMode("none");
    }
  };

  // Update telemetry mode of selected repo's service
  const onUpdateTelemetryMode = async () => {
    if (!selectedRepo) return;
    setUpdatingMode(true);
    try {
      const svcName = serviceInfo?.name || selectedRepo.name;
      await axios.put(
        API_ENDPOINTS.TOGGLES(selectedRepo.id, svcName, "dev"),
        { telemetry_mode: selectedMode }
      );
      alert("Telemetry mode updated successfully!");
    } catch (error) {
      console.error("Failed to update telemetry mode:", error);
      alert("Failed to update telemetry mode.");
    } finally {
      setUpdatingMode(false);
    }
  };

  // Determine if option should be disabled (for scan result)
  const isOptionDisabled = (mode: string) => {
    const info = scanResult || serviceInfo;
    if (!info) return false;

    const { has_metrics, has_otel } = info;

    // Both metrics and traces already exist
    if (has_metrics && has_otel) {
      return mode !== "none"; // Only allow "none"
    }

    // Has metrics but no traces
    if (has_metrics && !has_otel) {
      return mode === "metrics" || mode === "both";
    }

    // Has traces but no metrics
    if (!has_metrics && has_otel) {
      return mode === "traces" || mode === "both";
    }

    // Has neither - nothing disabled
    return false;
  };

  const getDisabledReason = (mode: string) => {
    const info = scanResult || serviceInfo;
    if (!info) return "";

    const { has_metrics, has_otel } = info;

    if (has_metrics && has_otel) {
      return "Already has both metrics and traces";
    }
    if (has_metrics && (mode === "metrics" || mode === "both")) {
      return "Already has metrics instrumentation";
    }
    if (has_otel && (mode === "traces" || mode === "both")) {
      return "Already has trace instrumentation";
    }
    return "";
  };

  return (
    <Box sx={{ p: 0 }}>
      <Typography variant="h6" fontWeight={700} mb={2}>
        Repo import
      </Typography>

      {/* Import Form */}
      <Box sx={{ display: "flex", gap: 2, mb: 2 }}>
        <TextField
          fullWidth
          size="small"
          variant="outlined"
          placeholder="https://github.com/user/project.git"
          value={githubURL}
          onChange={(e) => {
            setGithubURL(e.target.value);
            setShowScanResult(false); // Reset scan when URL changes
            setScanResult(null);
          }}
        />
        <Button
          variant="outlined"
          color="primary"
          onClick={handleScan}
          disabled={!githubURL || scanning}
          sx={{ minWidth: 110, fontWeight: 700, borderRadius: 2 }}
        >
          {scanning ? "Scanning..." : "Scan"}
        </Button>
      </Box>

      {/* Scan Result */}
      {showScanResult && scanResult && (
        <Card variant="outlined" sx={{ mb: 3, borderRadius: 2 }}>
          <CardContent>
            <Typography variant="h6" fontWeight={700} mb={2}>
              Scan Results
            </Typography>

            <Box display="flex" gap={1} mb={3}>
              <Chip
                label={`Framework: ${scanResult.framework}`}
                size="small"
                color="primary"
                variant="outlined"
              />
              <Chip
                icon={scanResult.has_metrics ? <CheckCircleIcon /> : <CancelIcon />}
                label={scanResult.has_metrics ? "Has Metrics" : "No Metrics"}
                size="small"
                color={scanResult.has_metrics ? "success" : "default"}
              />
              <Chip
                icon={scanResult.has_otel ? <CheckCircleIcon /> : <CancelIcon />}
                label={scanResult.has_otel ? "Has Traces" : "No Traces"}
                size="small"
                color={scanResult.has_otel ? "info" : "default"}
              />
            </Box>

            <FormControl component="fieldset" sx={{ mb: 2, width: "100%" }}>
              <FormLabel component="legend" sx={{ fontWeight: 700, mb: 2 }}>
                Select Telemetry Mode to Add
              </FormLabel>
              <RadioGroup
                value={selectedMode}
                onChange={(e) => setSelectedMode(e.target.value)}
              >
                {telemetryModes.map((tm) => {
                  const disabled = isOptionDisabled(tm.value);
                  const reason = getDisabledReason(tm.value);

                  return (
                    <FormControlLabel
                      key={tm.value}
                      value={tm.value}
                      control={<Radio />}
                      disabled={disabled}
                      label={
                        <Box>
                          <Typography
                            fontWeight={600}
                            variant="body1"
                            color={disabled ? "text.disabled" : "text.primary"}
                          >
                            {tm.label}
                          </Typography>
                          <Typography
                            variant="caption"
                            color={disabled ? "text.disabled" : "text.secondary"}
                            sx={{ pl: 0.5 }}
                          >
                            {disabled ? reason : tm.description}
                          </Typography>
                        </Box>
                      }
                      sx={{ alignItems: "flex-start", mb: 1.5 }}
                    />
                  );
                })}
              </RadioGroup>
            </FormControl>

            <Button
              variant="contained"
              color="primary"
              onClick={handleImport}
              disabled={loading}
              sx={{ fontWeight: 700 }}
            >
              {loading ? "Importing..." : "Import with Selected Mode"}
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Search Existing Repos */}
      <TextField
        fullWidth
        size="small"
        variant="outlined"
        placeholder="Search repositories..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        InputProps={{
          startAdornment: (
            <InputAdornment position="start">
              <SearchIcon sx={{ color: "grey.500" }} />
            </InputAdornment>
          ),
        }}
        sx={{ mb: 2 }}
      />

      <Paper
        variant="outlined"
        sx={{
          borderRadius: 2,
          overflow: "hidden",
          mb: 3,
          bgcolor: "#f9fafb",
          borderColor: "#e5e7eb",
          minHeight: 100,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        {fetchingRepos ? (
          <CircularProgress size={40} />
        ) : filteredRepos.length === 0 ? (
          <Box p={2} textAlign="center" color="text.secondary">
            No repositories found.
          </Box>
        ) : (
          <List dense={true} sx={{ width: "100%" }}>
            {filteredRepos.map((repo) => (
              <ListItem
                key={repo.id}
                sx={{
                  p: 0,
                  bgcolor:
                    selectedRepo?.id === repo.id ? "action.selected" : "inherit",
                }}
              >
                <ListItemButton onClick={() => onSelectRepo(repo)}>
                  <ListItemIcon>
                    <FolderIcon color="primary" />
                  </ListItemIcon>
                  <ListItemText
                    primary={repo.name}
                    secondary={
                      <span style={{ fontSize: "0.9em", color: "#888" }}>
                        {repo.id}
                      </span>
                    }
                  />
                </ListItemButton>
              </ListItem>
            ))}
          </List>
        )}
      </Paper>

      {/* Edit Existing Repo */}
      {selectedRepo && (
        <Card variant="outlined" sx={{ borderRadius: 2, borderColor: "#e5e7eb" }}>
          <CardContent>
            <Box mb={2}>
              <Typography variant="h6" fontWeight={700} mb={1}>
                {selectedRepo.name}
              </Typography>
              {serviceInfo && (
                <Box display="flex" gap={1} mb={2}>
                  <Chip
                    label={`Framework: ${serviceInfo.framework}`}
                    size="small"
                    color="primary"
                    variant="outlined"
                  />
                  {serviceInfo.has_metrics && (
                    <Chip
                      icon={<CheckCircleIcon />}
                      label="Has Metrics"
                      size="small"
                      color="success"
                    />
                  )}
                  {serviceInfo.has_otel && (
                    <Chip
                      icon={<CheckCircleIcon />}
                      label="Has Traces"
                      size="small"
                      color="info"
                    />
                  )}
                </Box>
              )}
            </Box>

            <FormControl component="fieldset" sx={{ mb: 2, width: "100%" }}>
              <FormLabel component="legend" sx={{ fontWeight: 700, mb: 2 }}>
                Update Telemetry Mode
              </FormLabel>
              <RadioGroup
                value={selectedMode}
                onChange={(e) => setSelectedMode(e.target.value)}
              >
                {telemetryModes.map((tm) => {
                  const disabled = isOptionDisabled(tm.value);
                  const reason = getDisabledReason(tm.value);

                  return (
                    <FormControlLabel
                      key={tm.value}
                      value={tm.value}
                      control={<Radio />}
                      disabled={disabled}
                      label={
                        <Box>
                          <Typography
                            fontWeight={600}
                            variant="body1"
                            color={disabled ? "text.disabled" : "text.primary"}
                          >
                            {tm.label}
                          </Typography>
                          <Typography
                            variant="caption"
                            color={disabled ? "text.disabled" : "text.secondary"}
                            sx={{ pl: 0.5 }}
                          >
                            {disabled ? reason : tm.description}
                          </Typography>
                        </Box>
                      }
                      sx={{ alignItems: "flex-start", mb: 1.5 }}
                    />
                  );
                })}
              </RadioGroup>
            </FormControl>
            <Button
              variant="contained"
              color="primary"
              onClick={onUpdateTelemetryMode}
              disabled={updatingMode}
              sx={{ fontWeight: 700 }}
            >
              {updatingMode ? "Updating..." : "Update Telemetry Mode"}
            </Button>
          </CardContent>
        </Card>
      )}
    </Box>
  );
}
