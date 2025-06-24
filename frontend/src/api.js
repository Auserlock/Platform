import axios from 'axios';

const apiClient = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
});

apiClient.interceptors.response.use(
  response => response.data,
  error => {
    console.error('API Error:', error.response || error.message);
    return Promise.reject(error);
  }
);

// --- API 函数定义 ---

/**
 * 获取所有任务列表
 * @returns {Promise<Array>} 任务数组
 */
export const getTasks = () => apiClient.get('/tasks');

/**
 * 根据ID获取单个任务的详细信息
 * @param {string} taskId 任务ID
 * @returns {Promise<Object>} 任务对象
 */
export const getTaskById = (taskId) => apiClient.get(`/tasks/${taskId}`);

/**
 * 根据ID删除一个任务
 * @param {string} taskId 任务ID
 */
export const deleteTask = (taskId) => apiClient.delete(`/tasks/${taskId}`);

/**
 * 提交一个新任务
 * @param {Object} data - 包含表单数据的对象
 * @returns {Promise<Object>} 创建成功后的新任务对象
 */
export const submitTask = (data) => {
  const formData = new FormData();
  const taskTypeForApi = data.taskType === 'build' ? 'kernel-build' : data.taskType;
  formData.append('task_type', taskTypeForApi);

  if (data.taskType === 'build') {
    if (data.jsonFile) {
      formData.append('report', data.jsonFile);
    } else if (data.jsonInput) {
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
  
  return apiClient.post('/tasks', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
};

/**
 * 下载任务产物 (Artifact)
 * GET /api/v1/artifacts/:id
 * @param {string} taskId - 任务的ID，用于构建下载链接。
 * @param {string} taskName - 任务的名称，用作默认下载文件名。
 */
export const downloadTaskArtifact = async (taskId, taskName) => {
  try {
    console.log(`开始下载任务产物，任务ID: ${taskId}`);
    
    // 创建一个不带拦截器的axios实例，专门用于文件下载，并设置响应类型为blob
    const downloadClient = axios.create({
      baseURL: '/api/v1',
      responseType: 'blob',
    });

    const response = await downloadClient.get(`/artifacts/${taskId}`);
    
    const blob = new Blob([response.data]);
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;

    // 尝试从响应头中获取服务器建议的文件名
    let filename = `${taskName.replace(/[\s/\\?%*:"|<>]/g, '_')}_artifact.zip`; // 安全的默认文件名
    const contentDisposition = response.headers['content-disposition'];
    if (contentDisposition) {
      const filenameMatch = contentDisposition.match(/filename="?(.+)"?/);
      if (filenameMatch && filenameMatch.length === 2) {
        filename = filenameMatch[1];
      }
    }
    
    link.setAttribute('download', filename);
    document.body.appendChild(link);
    link.click();
    
    // 清理
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);

  } catch (error) {
    console.error('产物下载失败:', error);
    // 尝试将返回的blob数据作为文本读取，看是否包含错误信息
    if (error.response && error.response.data) {
      const reader = new FileReader();
      reader.onload = function() {
        try {
          const errorJson = JSON.parse(this.result);
          alert(`文件下载失败: ${errorJson.error || '未知错误'}`);
        } catch (e) {
          alert(`文件下载失败: 无法解析服务器返回的错误信息。`);
        }
      };
      reader.readAsText(error.response.data);
    } else {
      alert('文件下载失败，请检查网络或联系管理员。');
    }
  }
};