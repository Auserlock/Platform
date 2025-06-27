import axios from 'axios';

// 创建一个全局的 axios 实例，用于大多数 API 请求
// 根据你的 Gin 日志，所有 API 路由都以 /api/v1 开头
const apiClient = axios.create({
  // 我们将 baseURL 设置为 /api/v1。
  // 在 Vite 配置中，你可以设置 proxy 将 /api/v1 转发到你的后端服务器 (例如 http://localhost:8080)
  // 这样可以避免跨域问题。
  baseURL: '/api/v1', 
  timeout: 10000, // 请求超时时间设置为 10 秒
});

// 添加响应拦截器
// 这是一个非常好的实践，可以统一处理所有响应和错误
apiClient.interceptors.response.use(
  // 对于成功的响应 (状态码在 2xx 范围内)，我们直接返回响应体中的 data 部分
  // 这样在调用 API 函数时，可以直接通过 .then(data => ...) 获取数据，无需 response.data
  response => response.data,
  
  // 对于失败的响应，我们在这里统一处理错误
  error => {
    // 在控制台打印详细的错误信息，便于调试
    console.error('API Error:', error.response || error.message);
    // 使用 Promise.reject 将错误继续传递下去，这样调用处的 .catch() 才能捕获到错误
    return Promise.reject(error);
  }
);

// --- API 函数定义 ---

/**
 * 获取所有任务列表
 * GET /api/v1/tasks
 * @returns {Promise<Array>} 任务对象组成的数组
 */
export const getTasks = () => apiClient.get('/tasks');

/**
 * 根据ID获取单个任务的详细信息
 * GET /api/v1/tasks/:id
 * @param {string} taskId - 任务ID
 * @returns {Promise<Object>} 单个任务对象
 */
export const getTaskById = (taskId) => apiClient.get(`/tasks/${taskId}`);

/**
 * 根据ID删除一个任务
 * DELETE /api/v1/tasks/:id
 * @param {string} taskId - 任务ID
 * @returns {Promise<any>}
 */
export const deleteTask = (taskId) => apiClient.delete(`/tasks/${taskId}`);

/**
 * 提交一个新任务
 * POST /api/v1/tasks
 * @param {Object} data - 包含表单数据的对象
 * @returns {Promise<Object>} 创建成功后的新任务对象
 */
export const submitTask = (data) => {
  const formData = new FormData();
  
  // 根据前端的 taskType 映射到后端需要的 task_type
  const taskTypeForApi = data.taskType === 'build' ? 'kernel-build' : data.taskType;
  formData.append('task_type', taskTypeForApi);

  // 根据不同的任务类型，附加不同的数据
  if (data.taskType === 'build') {
    if (data.jsonFile) {
      formData.append('report', data.jsonFile);
    } else if (data.jsonInput) {
      // 如果是手动输入的 JSON 字符串，则创建一个 Blob 对象再附加
      const jsonBlob = new Blob([data.jsonInput], { type: 'application/json' });
      formData.append('report', jsonBlob, 'config.json');
    }
    if (data.patchFile) {
      formData.append('patchFile', data.patchFile);
    }
  } else if (data.taskType === 'patch') {
    if (data.patchFile) {
      formData.append('patchFile', data.patchFile);
    }
    if (data.patchTargetTaskId) {
      formData.append('targetTaskId', data.patchTargetTaskId);
    }
  }
  
  // 发送 POST 请求，注意需要设置正确的 Content-Type
  return apiClient.post('/tasks', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
};

/**
 * 下载任务产物 (Artifact)
 * GET /api/v1/artifacts/:id
 * @param {string} artifactId - 产物的ID (根据 Gin 日志，这里应该是产物ID，可能与任务ID相同或不同)
 * @param {string} taskName - 任务的名称，用作默认下载文件名
 */
export const downloadTaskArtifact = (artifactId, taskName = 'download') => {
  const url = `/api/v1/artifacts/${artifactId}`;
  const a = document.createElement('a');
  a.href = url;
  a.download = ''; // 让后端通过 Content-Disposition 设置文件名
  a.style.display = 'none';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
};
