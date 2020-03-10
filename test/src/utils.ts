import WebSocket from 'ws'
// TODO: возможно надо юзать dc-config
const SERVER_ENDPOINT = process.env.SERVER_ENDPOINT || 'ws://localhost:8888/'
export const OPEN_STATE = 1

export const createConnection = (url: string = SERVER_ENDPOINT): Promise<WebSocket> => new Promise((resolve, reject) => {
    const client = new WebSocket(url)
    client.on('open', () => {
        resolve(client)
    })
    client.on('error', error => {
        reject(error)
    })
})

export const randomString = () =>
    Math.random()
        .toString(36)
        .substring(2, 15) +
    Math.random()
        .toString(36)
        .substring(2, 15)