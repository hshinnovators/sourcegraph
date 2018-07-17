import * as assert from 'assert'
import { MessageTransports } from '../jsonrpc2/connection'
import { Trace } from '../jsonrpc2/trace'
import { ClientCapabilities, InitializeParams, InitializeRequest, InitializeResult } from '../protocol'
import { Connection as ServerConnection, createConnection as createServerConnection } from '../server/server'
import { clientStateIsActive, createMessageTransports, getClientState } from '../test/integration/helpers'
import { Client, ClientState } from './client'

const createClientTransportsForTestServer = (registerServer: (server: ServerConnection) => void): MessageTransports => {
    const [clientTransports, serverTransports] = createMessageTransports()
    const serverConnection = createServerConnection(serverTransports)
    serverConnection.listen()
    registerServer(serverConnection)
    return clientTransports
}

describe('Client', () => {
    it('registers features, activates, initializes, stops, and reactivates', async () => {
        const initResult: InitializeResult = { capabilities: { hoverProvider: true } }
        const testNotificationParams = { a: 1 }
        const testRequestParams = { b: 2 }
        const testRequestResult = { c: 3 }

        // Create test server.
        let serverInitialized!: Promise<void>
        let serverReceivedTestNotification!: Promise<void>
        let serverReceivedTestRequest!: Promise<void>
        const createMessageTransports = () =>
            createClientTransportsForTestServer(server => {
                serverInitialized = new Promise<void>((resolve, reject) => {
                    server.onRequest(InitializeRequest.type, params => {
                        try {
                            assert.deepStrictEqual(params, {
                                root: null,
                                capabilities: { experimental: 'test' },
                                trace: Trace.toString(Trace.Off),
                                workspaceFolders: null,
                                initializationOptions: 'test',
                            } as InitializeParams)
                            resolve()
                        } catch (err) {
                            reject(err)
                        }
                        return initResult
                    })
                })
                serverReceivedTestNotification = new Promise<void>((resolve, reject) => {
                    server.onNotification('test', params => {
                        try {
                            assert.deepStrictEqual(params, testNotificationParams)
                            resolve()
                        } catch (err) {
                            reject(err)
                        }
                    })
                })
                serverReceivedTestRequest = new Promise<void>((resolve, reject) => {
                    server.onRequest('test', params => {
                        try {
                            assert.deepStrictEqual(params, testRequestParams)
                            resolve()
                        } catch (err) {
                            reject(err)
                        }
                        return testRequestResult
                    })
                })
            })

        const checkClient = async (client: Client): Promise<void> => {
            assert.strictEqual(getClientState(client), ClientState.Connecting)

            await Promise.all([clientStateIsActive(client), serverInitialized])
            assert.deepStrictEqual(client.initializeResult, initResult)

            client.sendNotification('test', testNotificationParams)
            await serverReceivedTestNotification

            await client.sendRequest('test', testRequestParams)
            await serverReceivedTestRequest

            client.onNotification('test', () => void 0)
            client.onRequest('test', () => void 0)

            assert.ok(client.needsStop())
            client.trace = Trace.Messages
            client.trace = Trace.Verbose
            client.trace = Trace.Off
        }

        // Create test client.
        const client = new Client('', '', { root: null, createMessageTransports })
        client.registerFeature({
            fillInitializeParams: (params: InitializeParams) => (params.initializationOptions = 'test'),
            fillClientCapabilities: (capabilities: ClientCapabilities) => (capabilities.experimental = 'test'),
            initialize: () => void 0,
        })
        assert.strictEqual(getClientState(client), ClientState.Initial)

        // Activate client.
        client.activate()
        assert.strictEqual(client.initializeResult, null)
        await checkClient(client)

        // Stop client and check that it reports itself as being stopped.
        await client.stop()
        assert.strictEqual(getClientState(client), ClientState.Stopped)
        assert.strictEqual(client.needsStop(), false)

        // Stop client again (noop because the client is already stopped).
        await client.stop()
        assert.strictEqual(getClientState(client), ClientState.Stopped)

        // Reactivate client.
        client.activate()
        await checkClient(client)

        client.unsubscribe()
    })
})