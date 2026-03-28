<script>
  import { onMount } from 'svelte';
  import { GetAvailableDisks, StartForensicCopy, StartRawCopy, StopCopy, FormatTargetDisks } from '../wailsjs/go/main/App';
  import { EventsOn } from '../wailsjs/runtime/runtime';

  let disks = [];
  let source = null;
  let targets = [];
  let progress = {};
  let logs = [];
  let copying = false;
  let formatTargets = false;
  let copyMode = 'file';

  $: selectableDisks = disks.filter(d => d?.path && d.path.endsWith('\\'));
  $: physicalDisks = disks.filter(d => d?.path && d.path.toLowerCase().startsWith('\\\\.\\physicaldrive'));
  $: displayDisks = copyMode === 'raw' ? physicalDisks : selectableDisks;

  onMount(() => {
    EventsOn("log", (msg) => logs = [...logs, `[${new Date().toLocaleTimeString()}] ${msg}`]);
    EventsOn("copy-progress", (p) => progress = { ...progress, ...p });
    EventsOn("copy-complete", (payload) => {
      copying = false;
      if (payload) {
        progress = { ...progress, ...payload, status: 'Completed' };
      }
    });
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

  function setCopyMode(mode) {
    if (copyMode === mode) return;
    copyMode = mode;
    source = null;
    targets = [];
    formatTargets = false;
  }

  async function startCopy() {
    if (!source || targets.length === 0) {
      logs = [...logs, `[Warning] Select a source and at least one target.`];
      return;
    }

    if (copyMode === 'file' && formatTargets) {
      const targetNames = targets.map(t => t.name || t.path).join(', ');
      const confirmed = window.confirm(`Format the target drives before copying?\n\nTargets: ${targetNames}\n\nThis will permanently erase data.`);
      if (!confirmed) {
        return;
      }
      try {
        await FormatTargetDisks(targets.map(t => t.path), source.path);
      } catch (e) {
        logs = [...logs, `[Error] Format failed: ${e.message}`];
        return;
      }
    }

    copying = true;
    if (copyMode === 'raw') {
      await StartRawCopy(source.path, targets.map(t => t.path));
      return;
    }

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
      <h2>Copy Mode</h2>
      <div style="display: flex; gap: 12px; margin-bottom: 12px;">
        <label style="display: flex; align-items: center; gap: 6px;">
          <input type="radio" name="copyMode" checked={copyMode === 'file'} on:change={() => setCopyMode('file')} />
          <span>Logical volumes</span>
        </label>
        <label style="display: flex; align-items: center; gap: 6px;">
          <input type="radio" name="copyMode" checked={copyMode === 'raw'} on:change={() => setCopyMode('raw')} />
          <span>Physical disks (raw)</span>
        </label>
      </div>
      {#if copyMode === 'raw'}
        <div style="font-size: 12px; opacity: 0.7; margin-bottom: 8px;">
          Raw imaging requires admin rights and uses physical disks like \\.\PhysicalDrive0.
        </div>
      {/if}
    </div>

    <div class="section">
      <h2>Source Drive</h2>
      <div class="disk-list">
        {#each displayDisks as d}
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
        {#each displayDisks.filter(d => source && d.path !== source.path) as d}
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
      <label style="display: flex; align-items: center; gap: 8px; margin-bottom: 12px; opacity: {copyMode === 'raw' ? 0.5 : 1};">
        <input type="checkbox" bind:checked={formatTargets} disabled={copyMode === 'raw'} />
        <span>Format targets before copying</span>
      </label>
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
