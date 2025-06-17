<script>
  import { onMount } from 'svelte';
  import { GetAvailableDisks, StartForensicCopy, StopCopy } from '../wailsjs/go/main/App';
  import { EventsOn } from '../wailsjs/runtime/runtime';

  let disks = [];
  let source = null;
  let targets = [];
  let progress = {};
  let logs = [];
  let copying = false;

  onMount(() => {
    EventsOn("log", (msg) => logs = [...logs, `[${new Date().toLocaleTimeString()}] ${msg}`]);
    EventsOn("copy-progress", (p) => progress = p);
    EventsOn("copy-complete", () => copying = false);
    EventsOn("error", (err) => {
      copying = false;
      logs = [...logs, `[ERROR] ${err}`];
    });
    refreshDrives();
  });

  async function refreshDrives() {
    try {
      disks = await GetAvailableDisks();
      source = null;
      targets = [];
    } catch (e) {
      logs = [...logs, `[Error] Could not fetch drives: ${e.message}`];
    }
  }

  function selectSource(d) {
    source = d;
    targets = [];
  }

  function toggleTarget(d) {
    const exists = targets.find(t => t.path === d.path);
    if (exists) {
      targets = targets.filter(t => t.path !== d.path);
    } else {
      targets = [...targets, d];
    }
  }

  async function startCopy() {
    if (!source || targets.length === 0) {
      logs = [...logs, `[Warning] Select a source and at least one target.`];
      return;
    }
    copying = true;
    await StartForensicCopy(source.path, targets.map(t => t.path));
  }

  async function stopCopy() {
    copying = false;
    await StopCopy();
  }

  function formatBytes(bytes) {
    if (!bytes || bytes === 0) return '0 B';
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return (bytes / Math.pow(1024, i)).toFixed(2) + ' ' + sizes[i];
  }
</script>

<main class="container">
  <div class="sidebar">
    <div class="section">
      <h2>Source Drive</h2>
      <div class="disk-list">
        {#each disks as d}
          <div class="disk-item {source?.path === d.path ? 'selected' : ''}" on:click={() => selectSource(d)}>
            <div class="disk-name">{d.name}</div>
            <div class="disk-details">{d.path}<br>{formatBytes(d.size)}</div>
          </div>
        {/each}
      </div>
      <button class="button" on:click={refreshDrives}>Refresh Drives</button>
    </div>

    <div class="section">
      <h2>Target Drives</h2>
      <div class="disk-list">
        {#each disks.filter(d => source && d.path !== source.path) as d}
          <div class="disk-item {targets.find(t => t.path === d.path) ? 'selected' : ''}" on:click={() => toggleTarget(d)}>
            <div class="disk-name">{d.name}</div>
            <div class="disk-details">{d.path}<br>{formatBytes(d.size)}</div>
          </div>
        {/each}
      </div>
      <div class="target-selection">
        <div class="selected-targets">
          {#if targets.length === 0}
            <span style="opacity: 0.6;">No targets selected</span>
          {:else}
            {#each targets as t}
              <span class="target-tag">{t.name}</span>
            {/each}
          {/if}
        </div>
      </div>
    </div>

    <div class="section">
      <button class="button" on:click={startCopy} disabled={copying}>Start Imaging</button>
      <button class="button danger" on:click={stopCopy} disabled={!copying}>Stop</button>
    </div>
  </div>

  <div class="main-content">
    <div class="section">
      <h2><span class="status-indicator {progress.status === 'Copying' ? 'status-active' : progress.status === 'Completed' ? 'status-complete' : 'status-idle'}"></span> Imaging Progress</h2>
      <div class="progress-section">
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>{progress.status || 'Idle'}</span>
          <span>{(progress.progress || 0).toFixed(2)}%</span>
        </div>
        <div class="progress-bar">
          <div class="progress-fill" style="width: {(progress.progress || 0)}%"></div>
        </div>
        <div class="progress-info">
          <div class="info-item">
            <div class="info-label">Bytes Copied</div>
            <div class="info-value">{formatBytes(progress.bytesCopied)}</div>
          </div>
          <div class="info-item">
            <div class="info-label">Total Size</div>
            <div class="info-value">{formatBytes(progress.totalBytes)}</div>
          </div>
          <div class="info-item">
            <div class="info-label">Speed</div>
            <div class="info-value">{(progress.speed / 1024 / 1024 || 0).toFixed(1)} MB/s</div>
          </div>
          <div class="info-item">
            <div class="info-label">Time Remaining</div>
            <div class="info-value">{progress.timeRemaining || '--'} s</div>
          </div>
        </div>
      </div>
    </div>

    <div class="section">
      <h2>Hash Verification</h2>
      <div class="hash-section">
        <div class="hash-item">
          <div class="hash-label">MD5 Hash</div>
          <div class="hash-value">{progress.md5Hash || 'Not calculated'}</div>
        </div>
        <div class="hash-item">
          <div class="hash-label">SHA-256 Hash</div>
          <div class="hash-value">{progress.sha256Hash || 'Not calculated'}</div>
        </div>
      </div>
    </div>

    <div class="section">
      <h2>Activity Log</h2>
      <div class="log-section">
        {#each logs as log}
          <div class="log-entry">{log}</div>
        {/each}
      </div>
    </div>
  </div>
</main>

<style>
  @import url("./style.css");
</style>
