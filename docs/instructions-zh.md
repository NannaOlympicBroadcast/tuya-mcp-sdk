本文为您介绍如何创建并配置自定义 MCP 服务，并在智能体中完成其调试。

## 创建自定义 MCP 服务

1. 登录 [涂鸦开发者平台](https://platform.tuya.com/)。

2. 进入 [**MCP 管理**](https://platform.tuya.com/exp/ai/mcp) > **自定义 MCP 服务** 页面，单击 **添加自定义 MCP**。

   ![添加 MCP.png](https://images.tuyacn.com/content-platform/hestia/17551511690284dbfc0e5.png)

3. 在 **注册 MCP Server** 弹窗中，填写中、英文的服务名称和服务描述，上传图片作为 Icon，并单击 **确定** 保存。

    <img alt="注册 MCP" src="https://images.tuyacn.com/content-platform/hestia/17551590340a8b915f70e.png" width="350" >


## 配置自定义 MCP 服务
:::info
若您的自定义 MCP 服务需在多个数据中心部署，请确保各数据中心的服务版本、工具列表保持一致，以避免因兼容性问题导致智能体编排异常或功能失效。 
:::
1. 创建完成 MCP 服务后，会自动进入其 **服务详情** 页面。
2. 在 **服务接入配置管理** > **数据中心** 下，单击右侧的 **添加数据中心**，并根据实际需求选择数据中心。

    ![服务详情.png](https://images.tuyacn.com/content-platform/hestia/1755153201c61f0c22e10.png)

3. 在所选数据中心下，查看 **接入地址**、**Access ID** 和 **Access secret** 并将信息复制粘贴到本地。请注意这些参数会在之后运行 MCP 的 SDK 时用到（具体请参考 [Github](https://github.com/tuya/tuya-mcp-sdk.git) 源码中的 **README** 说明）。

    ![数据中心.png](https://images.tuyacn.com/content-platform/hestia/1755153835f3f5944f767.png)

## 基于 SDK 访问 MCP 服务

请前往 [Github](https://github.com/tuya/tuya-mcp-sdk.git) 下载 MCP 的 SDK 并阅读相关资料。

![下载 MCP SDK.png](https://images.tuyacn.com/content-platform/hestia/1755154107774314342c7.png)

## 运行并调试 MCP 服务

接下来，为确保您的自定义 MCP 服务可以正常运行，需要在智能体中运行并调试 MCP 服务。

### 运行并调试

1. 首先，在所选数据中心下，检查 MCP 服务的服务状态。

    ![服务状态.png](https://images.tuyacn.com/content-platform/hestia/17551544896d7c045a3e2.png)

2. 然后，在 **工具** 页面，查看您的 MCP 服务的可用工具。
    
    ![可用工具.png](https://images.tuyacn.com/content-platform/hestia/175515474443f80346d8a.png)

3. 接下来，在要测试的工具下，单击 **试运行**。
 
    <img alt="试运行" src="https://images.tuyacn.com/content-platform/hestia/17551549220df8d41626d.png" width="700" >

4. 在 **试运行** 窗口，单击 **运行**，当左下角显示为 **调试通过** 时，则为 MCP 工具调试成功。

    <img alt="运行" src="https://images.tuyacn.com/content-platform/hestia/175515516744447009363.png" width="600" >


### 在智能体中添加 MCP 服务

1. 前往 [**我的智能体**](https://platform.tuya.com/exp/ai) 页面，单击 **开发版本** 进入智能体的开发页面。
2. 在 **01 模型能力配置** > **技能配置** 下，选择 **MCP 服务**，并单击右侧的添加（**+**）按钮。

    ![添加 MCP.png](https://images.tuyacn.com/content-platform/hestia/1755156952186f1bc3536.png)

3. 在 **添加 MCP 服务** 窗口 > **自定义 MCP 服务** 中，按需将 MCP 服务添加到智能体中。

    ![添加自定义 MCP 服务.png](https://images.tuyacn.com/content-platform/hestia/1755162525962dd48905c.png)

至此，您已经完成了自定义 MCP 服务的开发及调试过程。