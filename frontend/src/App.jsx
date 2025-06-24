import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import * as api from './api';
import {
  Upload,
  Send,
  Download,
  Terminal,
  FileText,
  Clock,
  CheckCircle,
  XCircle,
  AlertCircle,
  Play,
  Pause,
  RefreshCw,
  Eye,
  Trash2,
  Settings,
  Sun,
  Moon,
  ChevronDown,
  ChevronUp
} from 'lucide-react';

const LOCAL_STORAGE_GLOBAL_LOGS_KEY = 'kernelExperimentGlobalLogs';
const LOCAL_STORAGE_THEME_KEY = 'kernelExperimentTheme';
const LOCAL_STORAGE_EXPANDED_LOG_TASKS_KEY = 'kernelExperimentExpandedLogTasks';
const LOCAL_STORAGE_TERMINALS_KEY = 'kernelExperimentTerminals';

const formatDate = (dateString) => {
  if (!dateString) return '-';
  const d = new Date(dateString);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`;
};
const getStatusText = (status) => ({ running: '构建中', completed: '已完成', queued: '队列中', failed: '失败', success: '成功' }[status] || status);
const getStatusColorFromAPI = (status) => ({ completed: 'text-green-600 bg-green-100', success: 'text-green-600 bg-green-100', running: 'text-blue-600 bg-blue-100', queued: 'text-yellow-600 bg-yellow-100', failed: 'text-red-600 bg-red-100' }[status] || 'text-gray-600 bg-gray-100');
const getStatusIconFromAPI = (status) => ({ completed: <CheckCircle className="w-4 h-4" />, success: <CheckCircle className="w-4 h-4" />, running: <RefreshCw className="w-4 h-4 animate-spin" />, queued: <Clock className="w-4 h-4" />, failed: <XCircle className="w-4 h-4" /> }[status] || <AlertCircle className="w-4 h-4" />);


const KernelExperimentDashboard = () => {
  const getCurrentTime = () => new Date().toLocaleString('zh-CN');

  const [tasks, setTasks] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);

  const [globalLogs, setGlobalLogs] = useState(() => {
    try {
      const storedGlobalLogs = localStorage.getItem(LOCAL_STORAGE_GLOBAL_LOGS_KEY);
      return storedGlobalLogs ? JSON.parse(storedGlobalLogs) : [];
    } catch (error) {
      console.error("Failed to load global logs from localStorage:", error);
      return [];
    }
  });

  const [wsStatus, setWsStatus] = useState('connecting');
  const [activeTab, setActiveTab] = useState('tasks');
  const [jsonInput, setJsonInput] = useState('');
  const [jsonFile, setJsonFile] = useState(null);
  const [patchFile, setPatchFile] = useState(null);
  const [taskType, setTaskType] = useState('build');
  const [patchTargetTaskId, setPatchTargetTaskId] = useState('');

  const [theme, setTheme] = useState(() => {
    try {
      const storedTheme = localStorage.getItem(LOCAL_STORAGE_THEME_KEY);
      return storedTheme || 'light';
    } catch (error) {
      console.error("Failed to load theme from localStorage:", error);
      return 'light';
    }
  });

  const [expandedLogTasks, setExpandedLogTasks] = useState(() => {
    try {
      const storedExpanded = localStorage.getItem(LOCAL_STORAGE_EXPANDED_LOG_TASKS_KEY);
      return storedExpanded ? new Set(JSON.parse(storedExpanded)) : new Set();
    } catch (error) {
      console.error("Failed to load expanded log tasks from localStorage:", error);
      return new Set();
    }
  });

  const [qemuStatusTerminal, setQemuStatusTerminal] = useState(() => {
    try {
      const stored = localStorage.getItem(LOCAL_STORAGE_TERMINALS_KEY);
      const parsed = stored ? JSON.parse(stored).qemuStatus : null;
      return parsed || { output: 'QEMU状态终端初始化...\n', connected: false, input: '', connectedTaskId: '' };
    } catch (error) {
      console.error("Failed to load qemuStatusTerminal from localStorage:", error);
      return { output: 'QEMU状态终端初始化...\n', connected: false, input: '', connectedTaskId: '' };
    }
  });

  const [qemuSshTerminal, setQemuSshTerminal] = useState(() => {
    try {
      const stored = localStorage.getItem(LOCAL_STORAGE_TERMINALS_KEY);
      const parsed = stored ? JSON.parse(stored).qemuSsh : null;
      return parsed || { output: '尝试连接SSH到QEMU虚拟机...\nroot@kernel-test:~# ', connected: false, input: '', connectedTaskId: '' };
    } catch (error) {
      console.error("Failed to load qemuSshTerminal from localStorage:", error);
      return { output: '尝试连接SSH到QEMU虚拟机...\nroot@kernel-test:~# ', connected: false, input: '', connectedTaskId: '' };
    }
  });

  const [mcpTerminal, setMcpTerminal] = useState(() => {
    try {
      const stored = localStorage.getItem(LOCAL_STORAGE_TERMINALS_KEY);
      const parsed = stored ? JSON.parse(stored).mcp : null;
      return parsed || { output: 'MCP终端预留中...\n', connected: false, input: '', connectedTaskId: '' };
    } catch (error) {
      console.error("Failed to load mcpTerminal from localStorage:", error);
      return { output: 'MCP终端预留中...\n', connected: false, input: '', connectedTaskId: '' };
    }
  });
  
  const [taskIdToConnect, setTaskIdToConnect] = useState('');
  const [terminalTypeToConnect, setTerminalTypeToConnect] = useState('qemuSsh');

  const qemuStatusTerminalRef = useRef(null);
  const qemuSshTerminalRef = useRef(null);
  const mcpTerminalRef = useRef(null);

  useEffect(() => {
    try {
      localStorage.setItem(LOCAL_STORAGE_GLOBAL_LOGS_KEY, JSON.stringify(globalLogs));
    } catch (error) {
      console.error("Failed to save global logs to localStorage:", error);
    }
  }, [globalLogs]);

  useEffect(() => {
    try {
      localStorage.setItem(LOCAL_STORAGE_EXPANDED_LOG_TASKS_KEY, JSON.stringify(Array.from(expandedLogTasks)));
    } catch (error) {
      console.error("Failed to save expanded log tasks to localStorage:", error);
    }
  }, [expandedLogTasks]);

  useEffect(() => {
    try {
      localStorage.setItem(LOCAL_STORAGE_TERMINALS_KEY, JSON.stringify({
        qemuStatus: qemuStatusTerminal,
        qemuSsh: qemuSshTerminal,
        mcp: mcpTerminal,
      }));
    } catch (error) {
      console.error("Failed to save terminal states to localStorage:", error);
    }
  }, [qemuStatusTerminal, qemuSshTerminal, mcpTerminal]);

  useEffect(() => {
    try {
      localStorage.setItem(LOCAL_STORAGE_THEME_KEY, theme);
    } catch (error) {
      console.error("Failed to save theme to localStorage:", error);
    }
    document.documentElement.classList.remove('light', 'dark');
    document.documentElement.classList.add(theme);
    document.body.className = theme === 'dark' ? 'bg-gray-800' : 'bg-gray-100';
  }, [theme]);

  useEffect(() => {
    const connectWebSocket = () => {
      const wsProtocol = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
      const wsUrl = `${wsProtocol}${window.location.host}/api/v1/logs/ws`;
      console.log(`Connecting to WebSocket at: ${wsUrl}`);
      setWsStatus('connecting');
      const ws = new WebSocket(wsUrl);
      ws.onopen = () => {
        console.log('WebSocket connection established.');
        setWsStatus('connected');
      };
      ws.onmessage = (event) => {
        try {
          const newLog = JSON.parse(event.data);
          setGlobalLogs(prevLogs => [newLog, ...prevLogs]);
        } catch (e) {
          console.error('Failed to parse incoming log message:', event.data, e);
        }
      };
      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
      };
      ws.onclose = () => {
        console.log('WebSocket connection closed. Retrying in 5 seconds...');
        setWsStatus('disconnected');
        setTimeout(connectWebSocket, 5000);
      };
      return () => {
        ws.onclose = null;
        ws.close();
      };
    };
    const cleanup = connectWebSocket();
    return cleanup;
  }, []);

  useEffect(() => {
    if (qemuStatusTerminalRef.current) {
      qemuStatusTerminalRef.current.scrollTop = qemuStatusTerminalRef.current.scrollHeight;
    }
  }, [qemuStatusTerminal.output]);

  useEffect(() => {
    if (qemuSshTerminalRef.current) {
      qemuSshTerminalRef.current.scrollTop = qemuSshTerminalRef.current.scrollHeight;
    }
  }, [qemuSshTerminal.output]);

  useEffect(() => {
    if (mcpTerminalRef.current) {
      mcpTerminalRef.current.scrollTop = mcpTerminalRef.current.scrollHeight;
    }
  }, [mcpTerminal.output]);


  const getStatusColor = (status) => {
    switch (status) {
      case 'completed': return 'text-green-600 bg-green-100';
      case 'building': return 'text-blue-600 bg-blue-100';
      case 'queued': return 'text-yellow-600 bg-yellow-100';
      case 'failed': return 'text-red-600 bg-red-100';
      default: return 'text-gray-600 bg-gray-100';
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'completed': return <CheckCircle className="w-4 h-4" />;
      case 'building': return <RefreshCw className="w-4 h-4 animate-spin" />;
      case 'queued': return <Clock className="w-4 h-4" />;
      case 'failed': return <XCircle className="w-4 h-4" />;
      default: return <AlertCircle className="w-4 h-4" />;
    }
  };

  const handleJsonFileUpload = (event) => {
    const file = event.target.files[0];
    if (file && file.type === 'application/json') {
      setJsonFile(file);
      setJsonInput('');
    } else {
      alert('请上传有效的 JSON 文件。');
      setJsonFile(null);
    }
  };

  const handlePatchFileUpload = (event) => {
    const file = event.target.files[0];
    setPatchFile(file);
  };

  const addLog = useCallback((taskId, level, message) => {
    const newLog = { time: getCurrentTime(), level, message, taskId };
    setGlobalLogs(prevLogs => [newLog, ...prevLogs]);
  }, []);

  const fetchTasks = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const fetchedTasks = await api.getTasks() || [];
      setTasks(fetchedTasks.sort((a, b) => new Date(b.created_at) - new Date(a.created_at)));
    } catch (err) {
      setError("无法从服务器加载任务列表。");
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTasks();
    const intervalId = setInterval(fetchTasks, 30000);
    return () => clearInterval(intervalId);
  }, [fetchTasks]);
  
  const handleSubmitTask = useCallback(async () => {
    let isValid = true;
    let jsonContentProvided = jsonFile || jsonInput;
    if (taskType === 'build') { if (!jsonContentProvided) { alert('请为构建任务提供 JSON 报告或上传文件。'); isValid = false; } if (jsonInput && !jsonFile) { try { JSON.parse(jsonInput); } catch (e) { alert('JSON 内容格式不正确。'); isValid = false; } }
    } else if (taskType === 'patch') { if (!patchFile) { alert('请为补丁任务上传 Patch 文件。'); isValid = false; } if (!patchTargetTaskId) { alert('请为补丁任务选择关联的任务ID。'); isValid = false; } }
    if (!isValid) return;

    setIsLoading(true);
    try {
      const dataToSubmit = { taskType, jsonInput, jsonFile, patchFile, patchTargetTaskId };
      const newTaskFromApi = await api.submitTask(dataToSubmit);
      alert(`任务已成功提交至后端！新任务ID: ${newTaskFromApi.id}`);
      setJsonInput(''); setJsonFile(null); setPatchFile(null); setPatchTargetTaskId('');
      setActiveTab('tasks');
      await fetchTasks();
    } catch (err) {
      alert(`任务提交失败: ${err.response?.data?.error || err.message}`);
      console.error("提交失败:", err);
    } finally {
      setIsLoading(false);
    }
  }, [jsonFile, jsonInput, patchFile, taskType, patchTargetTaskId, fetchTasks]);

  const handleDeleteTask = useCallback(async (taskIdToDelete) => {
    const taskToDelete = tasks.find(t => t.id === taskIdToDelete);
    if (!taskToDelete) {
      alert("错误：找不到要删除的任务。");
      return;
    }
    if (taskToDelete.status !== 'completed' && taskToDelete.status !== 'failed') {
      const statusText = getStatusText(taskToDelete.status);
      alert(`无法删除任务：当前状态为 "${statusText}"。\n只有在“已完成”或“失败”状态下才能删除。`);
      return;
    }
    const taskName = taskToDelete?.payload?.title || taskIdToDelete;
    if (window.confirm(`您确定要删除任务 "${taskName}" 吗？此操作不可撤销。`)) {
      try {
        await api.deleteTask(taskIdToDelete);
        alert('任务已从服务器删除!');
        await fetchTasks();
      } catch (err) {
        alert(`删除失败: ${err.response?.data?.error || err.message}`);
      }
    }
  }, [fetchTasks, tasks]);

  const handleDownloadArtifact = useCallback(async (taskId, taskName) => {
    addLog(taskId, 'INFO', `开始下载任务 ${taskId} 的产物...`);
    await api.downloadTaskArtifact(taskId, taskName);
  }, [addLog]);

  const handleClearLogs = useCallback(() => {
    if (window.confirm('您确定要清除所有日志吗？此操作不可撤销。')) {
      setGlobalLogs([]);
      localStorage.removeItem(LOCAL_STORAGE_GLOBAL_LOGS_KEY);
      setExpandedLogTasks(new Set());
      localStorage.removeItem(LOCAL_STORAGE_EXPANDED_LOG_TASKS_KEY);
      alert('所有日志已清除。');
    }
  }, []);

  const renderLogEntry = (log, index) => {
    let displayTime = log.time;
    let displayLevel = log.level;
    let displayMessage = log.message;
    
    const structuredLogRegex = /time="([^"]+)"\s+level=(\w+)\s+msg="([\s\S]*)"/;
    const match = log.message?.match(structuredLogRegex);

    if (match) {
      displayTime = formatDate(match[1]);
      displayLevel = match[2].toUpperCase();
      displayMessage = match[3];
    }

    const isDark = theme === 'dark';

    const timeColor = isDark ? 'text-gray-500' : 'text-gray-600';
    const taskIdColor = isDark ? 'text-purple-400' : 'text-purple-700';
    const messageColor = isDark ? 'text-gray-300' : 'text-black';

    const levelColorClass = (isDark ? {
        INFO: 'text-blue-400',
        DEBUG: 'text-gray-400',
        WARNING: 'text-yellow-400',
        ERROR: 'text-red-400',
        CRITICAL: 'text-red-500',
    } : {
        INFO: 'text-blue-700',
        DEBUG: 'text-gray-700',
        WARNING: 'text-orange-600',
        ERROR: 'text-red-700',
        CRITICAL: 'text-red-700 font-bold',
    })[displayLevel] || (isDark ? 'text-blue-400' : 'text-blue-700');
    
    const task = tasks.find(t => t.id === log.taskId);
    const taskName = task?.payload?.title;

    return (
      <div key={index} className="flex flex-wrap items-baseline font-mono text-xs">
        <span className={`${timeColor} mr-2`}>[{displayTime}]</span>
        <span className={`mr-2 font-bold ${levelColorClass}`}>
          [{displayLevel}]
        </span>
        {log.taskId && log.taskId !== 'system' && log.taskId !== 'terminal' && (
          <span className={`mr-2 ${taskIdColor}`}>
            [{taskName ? taskName : `task:${log.taskId.substring(0,8)}...`}]
          </span>
        )}
        <span className={`whitespace-pre-wrap ${messageColor}`}>{displayMessage}</span>
      </div>
    );
  };

  const handleTerminalCommand = useCallback((e, terminalType) => {
    if (e.key === 'Enter') {
      let terminalState, setTerminalState;
      switch (terminalType) {
        case 'qemuStatus': setTerminalState = setQemuStatusTerminal; terminalState = qemuStatusTerminal; break;
        case 'qemuSsh': setTerminalState = setQemuSshTerminal; terminalState = qemuSshTerminal; break;
        case 'mcp': setTerminalState = setMcpTerminal; terminalState = mcpTerminal; break;
        default: return;
      }
      const command = terminalState.input.trim();
      let newOutput = terminalState.output + command + '\n';
      let response = '';
      setTerminalState(prev => ({ ...prev, output: newOutput, input: '' }));
      setTimeout(() => {
        if (terminalType === 'qemuSsh') {
          if (!terminalState.connected) {
            if (command === 'ssh root@qemu-vm') { response = 'Password: '; setTerminalState(prev => ({ ...prev, output: prev.output + response, connected: true, connectedTaskId: terminalState.connectedTaskId || 'global_ssh' })); addLog('terminal', 'INFO', `SSH connected to QEMU VM.`); return;
            } else if (command === 'password') { response = 'Welcome to QEMU VM.\nroot@kernel-test:~# '; setTerminalState(prev => ({ ...prev, output: prev.output + response, connected: true })); addLog('terminal', 'INFO', `SSH login successful.`); return;
            } else { response = `ssh: connect to host qemu-vm port 22: Connection refused\nroot@kernel-test:~# `; }
          } else {
            switch (command) {
              case 'ls': response = 'kernel-build  patches  output  logs  test-results\n'; break;
              case 'uname -a': response = 'Linux kernel-test 5.15.0-custom #1 SMP Fri Jun 13 03:42:00 EDT 2025 x86_64 x86_64 x86_64 GNU/Linux\n'; break;
              case 'dmesg | tail': response = '[12345.678901] Custom patch applied successfully\n[12346.123456] Module loaded: test_driver\n[12347.789012] System ready for testing\n'; break;
              case 'exit': response = 'Connection to qemu-vm closed.\n'; setTerminalState(prev => ({ ...prev, output: prev.output + response, connected: false, connectedTaskId: '' })); addLog('terminal', 'INFO', `SSH disconnected from QEMU VM.`); return;
              default: response = command ? `${command}: command not found\n` : '';
            }
            response += 'root@kernel-test:~# ';
          }
        } else if (terminalType === 'qemuStatus') {
          switch (command) {
            case 'help': response = '可用命令: status, start, stop, reboot\n'; break;
            case 'status': response = terminalState.connected ? 'QEMU VM 运行中。\n' : 'QEMU VM 未运行。\n'; break;
            case 'start': if (!terminalState.connected) { response = '启动 QEMU VM...\n'; setTerminalState(prev => ({ ...prev, connected: true, output: prev.output + response, connectedTaskId: 'global_qemu_status' })); addLog('terminal', 'INFO', `QEMU VM started.`); } else { response = 'QEMU VM 已经在运行。\n'; } break;
            case 'stop': if (terminalState.connected) { response = '关闭 QEMU VM...\n'; setTerminalState(prev => ({ ...prev, connected: false, output: prev.output + response, connectedTaskId: '' })); addLog('terminal', 'INFO', `QEMU VM stopped.`); } else { response = 'QEMU VM 未运行。\n'; } break;
            default: response = command ? `未知命令: ${command}\n` : '';
          }
        } else if (terminalType === 'mcp') {
          response = 'MCP终端模拟: 该终端为未来MCP相关功能预留。\n' + (command ? `${command}: command not supported here\n` : '');
        }
        setTerminalState(prev => ({ ...prev, output: prev.output + response }));
        addLog('terminal', 'INFO', `Terminal (${terminalType}) command: "${command}"`);
      }, 500);
    }
  }, [qemuStatusTerminal, setQemuStatusTerminal, qemuSshTerminal, setQemuSshTerminal, mcpTerminal, setMcpTerminal, addLog]);

  const handleTerminalInputChange = useCallback((e, terminalType) => {
    switch (terminalType) {
      case 'qemuStatus': setQemuStatusTerminal(prev => ({ ...prev, input: e.target.value })); break;
      case 'qemuSsh': setQemuSshTerminal(prev => ({ ...prev, input: e.target.value })); break;
      case 'mcp': setMcpTerminal(prev => ({ ...prev, input: e.target.value })); break;
      default: break;
    }
  }, []);

  const toggleTheme = useCallback(() => { setTheme(prevTheme => prevTheme === 'light' ? 'dark' : 'light'); }, []);

  const toggleLogExpansion = useCallback((taskId) => {
    setExpandedLogTasks(prev => { const newSet = new Set(prev); if (newSet.has(taskId)) { newSet.delete(taskId); } else { newSet.add(taskId); } return newSet; });
  }, []);

  const groupedLogs = useMemo(() => {
    const logsByTaskId = new Map();
    globalLogs.forEach(log => {
      const id = log.taskId || 'system';
      if (!logsByTaskId.has(id)) {
        logsByTaskId.set(id, []);
      }
      logsByTaskId.get(id).push(log);
    });

    const taskGroups = tasks.map(task => ({
      taskId: task.id,
      taskName: task.payload.title || `Unnamed Task (${task.id.substring(0, 8)})`,
      logs: logsByTaskId.get(task.id) || [],
      type: 'task',
      latestLogTime: (logsByTaskId.get(task.id) && logsByTaskId.get(task.id).length > 0)
        ? new Date(logsByTaskId.get(task.id)[0].time).getTime()
        : new Date(task.created_at).getTime(),
    }));
    tasks.forEach(task => logsByTaskId.delete(task.id));

    const orphanGroups = [];
    logsByTaskId.forEach((logs, taskId) => {
      if (taskId !== 'system' && taskId !== 'terminal') {
        orphanGroups.push({
          taskId: taskId,
          taskName: `未知/已删除任务的日志 (${taskId.substring(0, 8)}...)`,
          logs: logs,
          type: 'other',
          latestLogTime: logs.length > 0 ? new Date(logs[0].time).getTime() : 0,
        });
      }
    });
    
    const systemGroup = logsByTaskId.has('system') ? [{ taskId: 'system', taskName: '系统消息', logs: logsByTaskId.get('system'), type: 'system', latestLogTime: logsByTaskId.get('system')[0] ? new Date(logsByTaskId.get('system')[0].time).getTime() : 0 }] : [];
    const terminalGroup = logsByTaskId.has('terminal') ? [{ taskId: 'terminal', taskName: '终端日志', logs: logsByTaskId.get('terminal'), type: 'terminal', latestLogTime: logsByTaskId.get('terminal')[0] ? new Date(logsByTaskId.get('terminal')[0].time).getTime() : 0 }] : [];

    const allGroups = [...taskGroups, ...orphanGroups, ...systemGroup, ...terminalGroup];
    allGroups.sort((a, b) => b.latestLogTime - a.latestLogTime);

    return allGroups;
  }, [globalLogs, tasks]);

  const TerminalWindow = ({ title, terminalState, setTerminalState, terminalType, terminalRef, minHeightClass = 'h-64' }) => (
    <div className={`rounded-lg shadow ${theme === 'light' ? 'bg-white' : 'bg-gray-900'}`}>
      <div className={`px-6 py-4 border-b flex flex-col sm:flex-row justify-between items-center space-y-2 sm:space-y-0 sm:space-x-4 ${theme === 'light' ? 'border-gray-200' : 'border-gray-700'}`}>
        <h2 className={`text-lg font-medium ${theme === 'light' ? 'text-gray-900' : 'text-gray-100'}`}>{title}</h2>
        <div className="flex items-center space-x-2">
          <div className={`flex items-center space-x-2 text-sm ${terminalState.connected ? 'text-green-600' : 'text-red-600'}`}>
            <div className={`w-2 h-2 rounded-full ${terminalState.connected ? 'bg-green-500' : 'bg-red-500'}`}></div>
            <span>{terminalState.connected ? '已连接' : '未连接'}{terminalState.connectedTaskId && ` (任务ID: ${terminalState.connectedTaskId})`}</span>
          </div>
          {(terminalType === 'qemuStatus' || terminalType === 'qemuSsh') && (
            <button
              onClick={() => {
                if (terminalState.connected) {
                  setTerminalState(prev => ({ ...prev, output: prev.output + `\n[${getCurrentTime()}] 连接已断开。\n`, connected: false, connectedTaskId: '' }));
                  addLog('terminal', 'INFO', `Terminal (${terminalType}) disconnected.`);
                } else {
                  if (terminalType === 'qemuStatus') { setTerminalState(prev => ({ ...prev, output: prev.output + `\n[${getCurrentTime()}] 尝试连接QEMU状态...\n`, connected: true, connectedTaskId: 'global_qemu_status' })); addLog('terminal', 'INFO', `Terminal (${terminalType}) connected.`);
                  } else if (terminalType === 'qemuSsh') { setTerminalState(prev => ({ ...prev, output: prev.output + `\n[${getCurrentTime()}] 请输入 'ssh root@qemu-vm' 尝试连接SSH。\n`, connected: false })); }
                }
              }}
              className={`inline-flex items-center px-3 py-1 border shadow-sm text-sm leading-4 font-medium rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 ${ terminalState.connected ? 'border-red-300 text-red-700 bg-red-50 hover:bg-red-100' : 'border-green-300 text-green-700 bg-green-50 hover:bg-green-100' }`}>
              {terminalState.connected ? <Pause className="w-4 h-4 mr-1" /> : <Play className="w-4 h-4 mr-1" />}{terminalState.connected ? '断开' : '连接'}
            </button>
          )}
        </div>
      </div>
      <div className="p-4">
        <div className="bg-gray-900 rounded-lg overflow-hidden">
          <div className="bg-gray-800 px-4 py-2 flex items-center space-x-2">
            <div className="w-3 h-3 bg-red-500 rounded-full"></div><div className="w-3 h-3 bg-yellow-500 rounded-full"></div><div className="w-3 h-3 bg-green-500 rounded-full"></div>
            <span className="text-gray-400 text-sm ml-4">{title.replace('终端', '')}{terminalState.connectedTaskId && ` (任务ID: ${terminalState.connectedTaskId})`}</span>
          </div>
          <div ref={terminalRef} className={`p-4 ${minHeightClass} overflow-y-auto font-mono text-sm text-green-400 bg-gray-900`}>
            <pre className="whitespace-pre-wrap">{terminalState.output}</pre>
            {terminalState.connected && (
              <div className="flex items-center">
                <input type="text" value={terminalState.input} onChange={(e) => handleTerminalInputChange(e, terminalType)} onKeyPress={(e) => handleTerminalCommand(e, terminalType)} className="bg-transparent border-none outline-none text-green-400 font-mono text-sm flex-1" placeholder="输入命令..." autoFocus={terminalType === 'qemuSsh' && terminalState.connected} />
                <span className="text-green-400 animate-pulse">_</span>
              </div>
            )}
          </div>
        </div>
        {!terminalState.connected && (
          <div className="mt-4 text-center text-gray-500">
            <Terminal className="w-12 h-12 mx-auto mb-2 text-gray-300" />
            <p>{terminalType === 'qemuSsh' && '点击“连接”或输入 "ssh root@qemu-vm" 尝试连接 SSH。'}{terminalType === 'qemuStatus' && '点击“连接”或输入 "start" 启动 QEMU VM。'}{terminalType === 'mcp' && '此终端为未来MCP相关功能预留。'}</p>
          </div>
        )}
      </div>
    </div>
  );

  const connectToTerminalByTaskId = useCallback(() => {
    const idToConnect = taskIdToConnect.trim();
    if (!idToConnect) { alert('请输入任务ID。'); return; }
    const targetTask = tasks.find(task => task.id === idToConnect);
    if (targetTask) {
      let setTargetTerminalState; let terminalTitle;
      switch (terminalTypeToConnect) {
        case 'qemuStatus': setTargetTerminalState = setQemuStatusTerminal; terminalTitle = 'QEMU状态'; break;
        case 'qemuSsh': setTargetTerminalState = setQemuSshTerminal; terminalTitle = 'QEMU SSH'; break;
        case 'mcp': setTargetTerminalState = setMcpTerminal; terminalTitle = 'MCP'; break;
        default: return;
      }
      setTargetTerminalState(prev => ({ ...prev, connected: true, connectedTaskId: idToConnect, output: prev.output + `\n[${getCurrentTime()}] 连接到任务 ${idToConnect} (${targetTask.payload?.title || targetTask.name}) 的${terminalTitle}终端...\n` + (terminalTypeToConnect === 'qemuSsh' ? 'root@kernel-test:~# ' : '') }));
      addLog('terminal', 'INFO', `Connected terminal (${terminalTypeToConnect}) to Task ${idToConnect}.`);
      alert(`已连接到任务 ${idToConnect} 的 ${terminalTitle} 终端。`);
    } else {
      alert(`未找到任务ID为 ${idToConnect} 的任务。`);
    }
  }, [taskIdToConnect, tasks, addLog, getCurrentTime]);

  return (
    <div className={`min-h-screen ${theme === 'light' ? 'bg-gray-100 text-gray-900' : 'bg-gray-800 text-gray-100'}`}>
      <header className={`shadow-sm border-b ${theme === 'light' ? 'bg-white' : 'bg-gray-900 border-gray-700'}`}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-4">
            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-2"><Terminal className="w-8 h-8 text-blue-600" /><h1 className={`text-2xl font-bold ${theme === 'light' ? 'text-gray-900' : 'text-gray-100'}`}>内核任务实验平台</h1></div>
            </div>
            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-2 text-sm">
                <div className={`w-2 h-2 rounded-full ${wsStatus === 'connected' ? 'bg-green-500' : wsStatus === 'connecting' ? 'bg-yellow-500 animate-pulse' : 'bg-red-500'}`}></div>
                <span className={`${theme === 'light' ? 'text-gray-600' : 'text-gray-300'}`}>
                  {wsStatus === 'connected' ? '日志流已连接' : wsStatus === 'connecting' ? '连接日志流...' : '日志流已断开'}
                </span>
              </div>
              <div className="flex items-center space-x-2 text-sm"><div className={`w-2 h-2 rounded-full ${isLoading ? 'bg-yellow-500 animate-pulse' : 'bg-green-500'}`}></div><span className={`${theme === 'light' ? 'text-gray-600' : 'text-gray-300'}`}>{isLoading ? '通信中...' : '后端服务在线'}</span></div>
              <button onClick={toggleTheme} className={`p-1 rounded-full ${theme === 'light' ? 'text-gray-500 hover:bg-gray-100' : 'text-gray-300 hover:bg-gray-700'}`} title={theme === 'light' ? '切换到暗色主题' : '切换到亮色主题'}>{theme === 'light' ? <Moon className="w-5 h-5" /> : <Sun className="w-5 h-5" />}</button>
              <Settings className={`w-5 h-5 ${theme === 'light' ? 'text-gray-400 hover:text-gray-600' : 'text-gray-500 hover:text-gray-300'}`} />
            </div>
          </div>
        </div>
      </header>
      <div className={`border-b ${theme === 'light' ? 'bg-white' : 'bg-gray-900 border-gray-700'}`}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <nav className="flex space-x-8">
                {[{ id: 'submit', label: '提交任务', icon: Send }, { id: 'tasks', label: '任务管理', icon: FileText }, { id: 'logs', label: '系统日志', icon: Eye }, { id: 'terminal', label: '虚拟终端', icon: Terminal }].map(({ id, label, icon: Icon }) => (
                    <button key={id} onClick={() => setActiveTab(id)} className={`flex items-center space-x-2 py-4 px-1 border-b-2 font-medium text-sm ${ activeTab === id ? 'border-blue-500 text-blue-600' : `border-transparent ${theme === 'light' ? 'text-gray-500 hover:text-gray-700 hover:border-gray-300' : 'text-gray-400 hover:text-gray-200 hover:border-gray-500'}` }`}>
                        <Icon className="w-4 h-4" /><span>{label}</span>
                    </button>
                ))}
            </nav>
        </div>
      </div>
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8" >
        {activeTab === 'submit' && (
          <div className="space-y-6">
            <div className={`rounded-lg shadow p-6 ${theme === 'light' ? 'bg-white' : 'bg-gray-900'}`}>
              <h2 className={`text-lg font-medium mb-4 ${theme === 'light' ? 'text-gray-900' : 'text-gray-100'}`}>提交新任务</h2>
              <div className="mb-6">
                <h3 className={`text-md font-medium mb-2 ${theme === 'light' ? 'text-gray-700' : 'text-gray-200'}`}>选择任务类型：</h3>
                <div className="flex space-x-4">
                  <label className="inline-flex items-center"><input type="radio" className="form-radio h-4 w-4 text-blue-600" name="taskType" value="build" checked={taskType === 'build'} onChange={(e) => setTaskType(e.target.value)} /><span className={`ml-2 ${theme === 'light' ? 'text-gray-700' : 'text-gray-200'}`}>构建内核</span></label>
                  <label className="inline-flex items-center"><input type="radio" className="form-radio h-4 w-4 text-blue-600" name="taskType" value="patch" checked={taskType === 'patch'} onChange={(e) => setTaskType(e.target.value)} /><span className={`ml-2 ${theme === 'light' ? 'text-gray-700' : 'text-gray-200'}`}>补丁应用</span></label>
                </div>
              </div>
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {taskType === 'build' ? (
                  <div className="space-y-4">
                    <h3 className={`text-md font-medium ${theme === 'light' ? 'text-gray-700' : 'text-gray-200'}`}>JSON 配置</h3>
                    <textarea value={jsonInput} onChange={(e) => { setJsonInput(e.target.value); setJsonFile(null); }} placeholder={`{\n  "kernel_version": "5.15.0",\n  ...\n}`} className={`w-full h-64 p-3 border rounded-md focus:ring-blue-500 focus:border-blue-500 font-mono text-sm ${theme === 'light' ? 'bg-white border-gray-300 text-gray-900' : 'bg-gray-800 border-gray-600 text-gray-100'}`} disabled={!!jsonFile} />
                    <div className={`border-2 border-dashed rounded-lg p-3 text-center transition-colors ${theme === 'light' ? 'border-gray-300 hover:border-gray-400' : 'border-gray-600 hover:border-gray-500'}`}>
                      <Upload className={`mx-auto h-8 w-8 ${theme === 'light' ? 'text-gray-400' : 'text-gray-500'}`} />
                      <div className="mt-2"><label htmlFor="json-upload" className="cursor-pointer"><span className={`block text-sm font-medium ${theme === 'light' ? 'text-gray-900' : 'text-gray-100'}`}>点击上传 JSON 文件</span><span className={`block text-xs ${theme === 'light' ? 'text-gray-500' : 'text-gray-400'}`}>支持 .json 格式</span></label><input id="json-upload" type="file" className="sr-only" accept=".json" onChange={handleJsonFileUpload} disabled={!!jsonInput} /></div>
                      {jsonFile && ( <div className="mt-2 text-sm text-blue-600">已选择: {jsonFile.name}</div> )}
                    </div>
                  </div>
                ) : (
                  <div className="space-y-4">
                    <h3 className={`text-md font-medium ${theme === 'light' ? 'text-gray-700' : 'text-gray-200'}`}>关联构建任务ID</h3>
                    <p className={`text-sm ${theme === 'light' ? 'text-gray-500' : 'text-gray-400'}`}>请选择或输入一个已存在的任务ID，此补丁将应用于该任务的环境。</p>
                    <div className="space-y-2"><label htmlFor="task-select" className={`block text-sm font-medium ${theme === 'light' ? 'text-gray-700' : 'text-gray-300'}`}>从列表中选择:</label><select id="task-select" value={patchTargetTaskId} onChange={(e) => setPatchTargetTaskId(e.target.value)} className={`block w-full pl-3 pr-10 py-2 text-base border rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm ${theme === 'light' ? 'bg-white border-gray-300 text-gray-900' : 'bg-gray-700 border-gray-600 text-gray-100'}`}><option value="">-- 请选择一个任务 --</option>{tasks.filter(t => t.payload).map(task => ( <option key={task.id} value={task.id}>{task.payload.title} ({task.id.substring(0,8)}...)</option> ))}</select></div>
                    <div className="space-y-2"><label htmlFor="task-input" className={`block text-sm font-medium ${theme === 'light' ? 'text-gray-700' : 'text-gray-300'}`}>或手动输入ID:</label><input type="text" id="task-input" value={patchTargetTaskId} onChange={(e) => setPatchTargetTaskId(e.target.value)} placeholder="手动输入任务ID" className={`w-full p-2 border rounded-md focus:ring-blue-500 focus:border-blue-500 text-sm ${theme === 'light' ? 'bg-white border-gray-300 text-gray-900' : 'bg-gray-800 border-gray-600 text-gray-100'}`} /></div>
                  </div>
                )}
                <div className="space-y-4">
                  <h3 className={`text-md font-medium ${theme === 'light' ? 'text-gray-700' : 'text-gray-200'}`}>Patch 文件</h3>
                  <div className={`border-2 border-dashed rounded-lg p-6 text-center transition-colors ${theme === 'light' ? 'border-gray-300 hover:border-gray-400' : 'border-gray-600 hover:border-gray-500'}`}>
                    <Upload className={`mx-auto h-12 w-12 ${theme === 'light' ? 'text-gray-400' : 'text-gray-500'}`} />
                    <div className="mt-4"><label htmlFor="patch-upload" className="cursor-pointer"><span className={`mt-2 block text-sm font-medium ${theme === 'light' ? 'text-gray-900' : 'text-gray-100'}`}>点击上传 Patch 文件</span><span className={`mt-1 block text-xs ${theme === 'light' ? 'text-gray-500' : 'text-gray-400'}`}>支持 .patch, .diff 格式</span></label><input id="patch-upload" type="file" className="sr-only" accept=".patch,.diff" onChange={handlePatchFileUpload} /></div>
                    {patchFile && ( <div className="mt-4 text-sm text-blue-600">已选择: {patchFile.name}</div> )}
                  </div>
                </div>
              </div>
              <div className="mt-6 flex justify-end">
                <button onClick={handleSubmitTask} disabled={ (taskType === 'build' && (!jsonInput && !jsonFile)) || (taskType === 'patch' && (!patchTargetTaskId || !patchFile)) || isLoading } className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:bg-gray-400 disabled:cursor-not-allowed"><Send className="w-4 h-4 mr-2" />{isLoading ? '提交中...' : '提交任务'}</button>
              </div>
            </div>
          </div>
        )}
        {activeTab === 'tasks' && (
          <div className="space-y-6">
            <div className={`rounded-lg shadow ${theme === 'light' ? 'bg-white' : 'bg-gray-900'}`}>
              <div className={`px-6 py-4 border-b flex justify-between items-center ${theme === 'light' ? 'border-gray-200' : 'border-gray-700'}`}>
                <h2 className={`text-lg font-medium ${theme === 'light' ? 'text-gray-900' : 'text-gray-100'}`}>任务列表</h2>
                <button onClick={fetchTasks} disabled={isLoading} title="刷新" className="p-1 rounded-full text-gray-500 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700 disabled:opacity-50">
                  <RefreshCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
                </button>
              </div>
              {isLoading && <div className="text-center p-8">正在从服务器加载任务...</div>}
              {error && <div className="text-center p-8 text-red-500">{error}</div>}
              {!isLoading && !error && (
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                    <thead className={`${theme === 'light' ? 'bg-gray-50' : 'bg-gray-800'}`}>
                      <tr>
                        <th className={`px-6 py-3 text-left text-xs font-medium uppercase tracking-wider ${theme === 'light' ? 'text-gray-500' : 'text-gray-400'}`}>任务ID</th>
                        <th className={`px-6 py-3 text-left text-xs font-medium uppercase tracking-wider ${theme === 'light' ? 'text-gray-500' : 'text-gray-400'}`}>名称</th>
                        <th className={`px-6 py-3 text-left text-xs font-medium uppercase tracking-wider ${theme === 'light' ? 'text-gray-500' : 'text-gray-400'}`}>状态</th>
                        <th className={`px-6 py-3 text-left text-xs font-medium uppercase tracking-wider ${theme === 'light' ? 'text-gray-500' : 'text-gray-400'}`}>创建时间</th>
                        <th className={`px-6 py-3 text-left text-xs font-medium uppercase tracking-wider ${theme === 'light' ? 'text-gray-500' : 'text-gray-400'}`}>完成时间</th>
                        <th className={`px-6 py-3 text-left text-xs font-medium uppercase tracking-wider ${theme === 'light' ? 'text-gray-500' : 'text-gray-400'}`}>操作</th>
                      </tr>
                    </thead>
                    <tbody className={`${theme === 'light' ? 'bg-white divide-gray-200' : 'bg-gray-900 divide-gray-700'}`}>
                      {tasks.map((task) => {
                        const isApiTask = !!task.payload;
                        const status = task.status;
                        const displayName = isApiTask ? task.payload.title : task.name;
                        const displayTime = isApiTask ? formatDate(task.created_at) : task.submitTime;
                        const completionTime = isApiTask ? formatDate(task.finished_at) : task.completeTime;
                        const statusColor = isApiTask ? getStatusColorFromAPI(status) : getStatusColor(status);
                        const statusIcon = isApiTask ? getStatusIconFromAPI(status) : getStatusIcon(status);
                        const statusText = isApiTask ? getStatusText(status) : (status === 'completed' ? '已完成' : status === 'building' ? '构建中' : '队列中');
                        return (
                          <tr key={task.id} className={`${theme === 'light' ? 'hover:bg-gray-50' : 'hover:bg-gray-800'}`}>
                            <td className={`px-6 py-4 whitespace-nowrap text-sm font-medium ${theme === 'light' ? 'text-gray-900' : 'text-gray-100'} font-mono`} title={task.id}>{task.id.substring(0,8)}...</td>
                            <td className={`px-6 py-4 whitespace-nowrap text-sm ${theme === 'light' ? 'text-gray-900' : 'text-gray-100'} max-w-xs truncate`} title={displayName}>{displayName}</td>
                            <td className="px-6 py-4 whitespace-nowrap">
                              <span className={`inline-flex items-center space-x-1 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor}`}>
                                {statusIcon}
                                <span>{statusText}</span>
                              </span>
                            </td>
                            <td className={`px-6 py-4 whitespace-nowrap text-sm ${theme === 'light' ? 'text-gray-500' : 'text-gray-300'}`}>{displayTime}</td>
                            <td className={`px-6 py-4 whitespace-nowrap text-sm ${theme === 'light' ? 'text-gray-500' : 'text-gray-300'}`}>{completionTime || '-'}</td>
                            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                              <div className="flex items-center space-x-2">
                                {(status === 'completed' || status === 'success') && ( <button onClick={() => handleDownloadArtifact(task.id, displayName)} className="text-blue-600 hover:text-blue-900" title="下载产物"><Download className="w-4 h-4" /></button> )}
                                <button onClick={() => handleDeleteTask(task.id)} className={`text-gray-400 hover:text-red-600 dark:text-gray-500 dark:hover:text-red-400`} title="删除任务"><Trash2 className="w-4 h-4" /></button>
                              </div>
                            </td>
                          </tr>
                        )
                      })}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        )}
        {activeTab === 'logs' && (
          <div className="space-y-6">
            <div className={`rounded-lg shadow ${theme === 'light' ? 'bg-white' : 'bg-gray-900'}`}>
              <div className={`px-6 py-4 border-b flex justify-between items-center ${theme === 'light' ? 'border-gray-200' : 'border-gray-700'}`}>
                <h2 className={`text-lg font-medium ${theme === 'light' ? 'text-gray-900' : 'text-white'}`}>系统日志</h2>
                <div className="flex items-center space-x-4">
                  <button className={`inline-flex items-center px-3 py-1 border shadow-sm text-sm leading-4 font-medium rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 ${theme === 'light' ? 'border-gray-300 text-gray-700 bg-white hover:bg-gray-50' : 'border-gray-600 text-gray-200 bg-gray-800 hover:bg-gray-700'}`} onClick={() => setExpandedLogTasks(new Set())}><RefreshCw className="w-4 h-4 mr-1" />全部折叠</button>
                  <button onClick={handleClearLogs} className="inline-flex items-center px-3 py-1 border border-transparent text-sm leading-4 font-medium rounded-md shadow-sm text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"><Trash2 className="w-4 h-4 mr-1" />清除所有日志</button>
                </div>
              </div>
              <div className={`p-4 ${theme === 'light' ? 'bg-gray-50' : 'bg-gray-800'}`}>
                {globalLogs.length > 0 ? (
                  <div className="space-y-2">
                    {groupedLogs.map(group => (
                      <div key={group.taskId} className={`border rounded-lg ${theme === 'light' ? 'border-gray-200 bg-white' : 'border-gray-700 bg-gray-900'}`}>
                        <button onClick={() => toggleLogExpansion(group.taskId)} className={`flex items-center justify-between w-full p-3 font-medium text-left ${theme === 'light' ? 'text-gray-700 hover:bg-gray-100' : 'text-gray-100 hover:bg-gray-700'}`}>
                          <span className="flex items-center">
                            {expandedLogTasks.has(group.taskId) ? <ChevronUp className="w-5 h-5 mr-2 text-blue-500" /> : <ChevronDown className="w-5 h-5 mr-2 text-gray-500" />}
                            <span className="font-semibold">{group.taskName}</span>
                          </span>
                          <span className={`text-xs ${theme === 'light' ? 'text-gray-500' : 'text-gray-400'}`}>{group.logs.length} 条日志</span>
                        </button>
                        {expandedLogTasks.has(group.taskId) && (
                          <div className={`p-3 border-t ${theme === 'light' ? 'border-gray-200 bg-white' : 'border-gray-700 bg-gray-900'}`}>
                            <div className="max-h-96 overflow-y-auto">
                              {group.logs.map((log, logIndex) => renderLogEntry(log, logIndex))}
                            </div>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                ) : ( <div className={`text-center py-8 ${theme === 'light' ? 'text-gray-500' : 'text-gray-400'}`}>等待从服务器接收日志...</div> )}
              </div>
            </div>
          </div>
        )}
        {activeTab === 'terminal' && (
          <div className="space-y-6">
            <div className={`rounded-lg shadow p-4 ${theme === 'light' ? 'bg-white' : 'bg-gray-900'}`}>
              <h3 className={`text-md font-medium mb-3 ${theme === 'light' ? 'text-gray-900' : 'text-gray-100'}`}>连接到任务终端</h3>
              <div className="flex flex-wrap items-center gap-2">
                <label htmlFor="task-id-input" className={`text-sm font-medium ${theme === 'light' ? 'text-gray-700' : 'text-gray-200'}`}>任务ID:</label>
                <input id="task-id-input" type="text" value={taskIdToConnect} onChange={(e) => setTaskIdToConnect(e.target.value)} placeholder="输入任务ID" className={`flex-grow min-w-[150px] p-2 border rounded-md focus:ring-blue-500 focus:border-blue-500 text-sm ${theme === 'light' ? 'bg-white border-gray-300 text-gray-900' : 'bg-gray-800 border-gray-600 text-gray-100'}`} />
                <label htmlFor="terminal-type-to-connect-select" className={`text-sm font-medium ${theme === 'light' ? 'text-gray-700' : 'text-gray-200'}`}>连接类型:</label>
                <select id="terminal-type-to-connect-select" value={terminalTypeToConnect} onChange={(e) => setTerminalTypeToConnect(e.target.value)} className={`block w-auto pl-3 pr-8 py-2 text-base border rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm ${theme === 'light' ? 'bg-white border-gray-300 text-gray-900' : 'bg-gray-700 border-gray-600 text-gray-100'}`}>
                  <option value="qemuSsh">QEMU SSH</option><option value="qemuStatus">QEMU状态</option><option value="mcp">MCP</option>
                </select>
                <button onClick={connectToTerminalByTaskId} className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:bg-gray-300 disabled:cursor-not-allowed" disabled={!taskIdToConnect}>连接</button>
              </div>
            </div>
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <div className="flex flex-col space-y-6">
                <TerminalWindow title="QEMU状态终端" terminalState={qemuStatusTerminal} setTerminalState={setQemuStatusTerminal} terminalType="qemuStatus" terminalRef={qemuStatusTerminalRef} minHeightClass="h-72" />
                <TerminalWindow title="MCP终端 (预留)" terminalState={mcpTerminal} setTerminalState={setMcpTerminal} terminalType="mcp" terminalRef={mcpTerminalRef} minHeightClass="h-72" />
              </div>
              <div className="flex-grow">
                <TerminalWindow title="QEMU SSH终端" terminalState={qemuSshTerminal} setTerminalState={setQemuSshTerminal} terminalType="qemuSsh" terminalRef={qemuSshTerminalRef} minHeightClass="h-[600px]" />
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
};

export default KernelExperimentDashboard;