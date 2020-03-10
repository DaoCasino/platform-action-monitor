import { expect } from 'chai'
import { OPEN_STATE, createConnection } from './utils'
import WebSocket from 'ws'

// dc-messaging  pubsub room test - там есть интеграционный тест на веб сокет сервер

describe('Platform Action Monitor smoke test', () => {
    describe('Connection', () => {
        let client:WebSocket
        it('When a client connects, then a connection is established', async () => {
            client = await createConnection()
            expect(client.readyState).to.equal(OPEN_STATE)
        })
        after(() => {
            // если сетевое соединение открывается, оно должно закрываться
            if (client && 'terminate' in client) {
                client.terminate()
            }
        })
    })
})