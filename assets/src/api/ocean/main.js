import requestBase from '../request-base'

export default function request(method, url, data, header) {  //暴露 request 给我们好API 管理


    if(process.env.NODE_ENV == 'development') {
        url = `//127.0.0.1:8888${url}`;
    } else {
        url = `//insight.biz.weibo.com${url}`;
    }

    return requestBase(method, url, data, {}, {
        dev : 'insight.biz.weibo.com',
        online : 'insight.biz.weibo.com'
    })
}