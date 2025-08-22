This topic describes how to create and configure a custom MCP server and debug it in an agent.

## Create custom MCP server

1. Log in to the [Tuya Developer Platform](https://platform.tuya.com/).

2. Go to [**MCP Management**](https://platform.tuya.com/exp/ai/mcp) > **Custom MCP Service**, and click **Add custom MCP**.

   ![Add MCP](https://images.tuyacn.com/content-platform/hestia/17552251294ef4fcc8a0a.png)

3. In the **Sign up MCP Server** dialog, enter the service name and description in Chinese and English, upload an image as the icon, and then click **Confirm** to save.

   <img alt="Sign up MCP server" src="https://images.tuyacn.com/content-platform/hestia/17552253115b7d3028283.png" width="350" >


## Configure custom MCP server
:::info
If your custom MCP service is deployed across multiple data centers, ensure consistent service versions and tool configurations in all data centers to prevent compatibility issues that may disrupt agent orchestration or cause functional failures.
:::
1. After creating the MCP server, you will automatically be redirected to its **Service Details** page.
2. In the section of **Service Access Configuration Management** > **Data Center**, click **Add Data Center** on the right, select a data center as needed, and then click **OK**.

   ![Service details](https://images.tuyacn.com/content-platform/hestia/1755225536581a0ae9478.png)

3. Click the selected data center, and you can see the **Endpoint**, **Access ID**, and **Access Secret**. Copy and paste the information to your local device. Please note that these parameters will be used when running the MCP SDK later. For more information, see the **README** in the [GitHub source code](https://github.com/tuya/tuya-mcp-sdk.git).

   ![Data center](https://images.tuyacn.com/content-platform/hestia/175522559927d1caf4f12.png)

## Access MCP server via SDK

Download the MCP SDK from [GitHub](https://github.com/tuya/tuya-mcp-sdk.git) and read the relevant documents.

![Download](https://images.tuyacn.com/content-platform/hestia/1755154107774314342c7.png)

## Run and debug MCP Server

To ensure your custom MCP server operates properly, follow these steps to run and debug it within the agent environment:

### Run and debug

1. In the selected data center, check the service status of the MCP server.

   ![Service status](https://images.tuyacn.com/content-platform/hestia/1755225694a9430fec3a3.png)

2. On the **Tool** tab, view the available tools for your MCP server.

   ![Available tools](https://images.tuyacn.com/content-platform/hestia/17552257539b261737630.png)

3. Click **Test Run** to test your desired tool.

   <img alt="Test run" src="https://images.tuyacn.com/content-platform/hestia/1755225801672df842cdd.png" width="" >

4. In the **Test Run** window, click **Run**. When **Commissioning passed** appear in the lower left corner, the MCP tool has been debugged successfully.

   <img alt="Run" src="https://images.tuyacn.com/content-platform/hestia/175522586669f31b0a863.png" width="" >


### Add server to agent

1. Go to the [**My Agent**](https://platform.tuya.com/exp/ai) page, click **Develop** in the **Operation** column.
2. In the section **01 Model Configuration** > **Skills Configuration**, find **MCP Service** and click **+** on the right.

   ![Add MCP](https://images.tuyacn.com/content-platform/hestia/175522598482331e349fe.png)

3. On the **Add MCP service** page, click **Custom MCP Service** and add the desired MCP server to your agent.

   ![Add custom MCP server](https://images.tuyacn.com/content-platform/hestia/17552261567736d8f24ef.png)

So far, you have completed the development and debugging process of a custom MCP server.