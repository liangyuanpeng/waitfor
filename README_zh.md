# waitfor  

waitfor是为了解决需要再一些场景下等待某个job完成才启动的问题,waitfor容器作为init容器配置,监听某个job处于完成状态才算init成功. 

# 已经使用的场景  

在[replacer](https://github.com/liangyuanpeng/replacer)当中使用waitfor作为init容器等待给webhook做patch操作的job.