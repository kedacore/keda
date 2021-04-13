import { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios'
import CreateClient from './openstack-helper'

interface SwiftObjectMetadata {
    bytes: string,
    last_modified: string,
    hash: string,
    name: string,
    content_type: string
}

interface SwiftBulkDeleteResponse {
    'Number Not Found': number,
    'Response Status': string,
    Errors: Array<any>,
    'Number Deleted': number,
    'Response Body': string
}

interface DeleteAllObjectsResponse {
    isEmpty: boolean,
    response: string
}

// Wrapper for Openstack Swift API methods
export default class SwiftClient {
    private static swiftClient: SwiftClient
    private static api: AxiosInstance

    private constructor() {};

    public static async create(swiftURL ?: string): Promise<SwiftClient> {
        try {
            if(!SwiftClient.swiftClient) {
                if (swiftURL) {
                    this.api = await CreateClient('swift', swiftURL)
                } else {
                    this.api = await CreateClient('swift')
                }

                this.swiftClient = new SwiftClient()
            }
            return this.swiftClient
        } catch (err) {
            console.log(err.message);
        }
    }

    // Get metadata about all objects inside a Swift container.
    // It returns:
    //      An array of SwiftObjectMetadata, if the query parameter "format" is set to 'json'
    //      A string containing the filenames separate by newline if "format" is set to 'plain'
    public async getObjectsMetadata(containerName: string, config?: AxiosRequestConfig): Promise<Array<SwiftObjectMetadata> | string> {
        const response: AxiosResponse<Array<SwiftObjectMetadata> | string> = await SwiftClient.api.get(containerName, config)

        return response.data
    }

    public async getObjectCount(containerName: string, onlyFiles = false, config?: AxiosRequestConfig): Promise<number> {
        const containerObjects = <Array<SwiftObjectMetadata>> await this.getObjectsMetadata(containerName, {
            ...config,
            params: {
                format: 'json'
            }
        })

        if (onlyFiles) {
            return containerObjects.filter(object => !object.name.endsWith('/')).length
        }

        return containerObjects.length
    }

    public async createObject(containerName: string, objectName: string, config?: AxiosRequestConfig): Promise<void> {
        await SwiftClient.api.put(containerName + '/' + objectName, config)
    }

    public async deleteObject(containerName: string, objectName: string, config?: AxiosRequestConfig): Promise<void> {
        await SwiftClient.api.delete(containerName + '/' + objectName, config)
    }

    public async deleteAllObjects(containerName: string): Promise<DeleteAllObjectsResponse> {
        const containerObjects = <string> await this.getObjectsMetadata(containerName, {
            params: {
                format: 'plain'
            }
        })

        const formattedObjects = containerObjects
            .replace(/\n$/, '')
            .replace(/^/gm, '/' + containerName + '/');

        const response: AxiosResponse<SwiftBulkDeleteResponse> = await SwiftClient.api.post('', formattedObjects, {
            headers: {
                'content-type': 'text/plain'
            },
            params: {
                'bulk-delete': '',
                'format': 'json'
            }
        })

        return { isEmpty: response.data['Number Not Found'] === 0, response: JSON.stringify(response.data) }
    }

    public async createContainer(containerName: string, config?: AxiosRequestConfig): Promise<void> {
        await SwiftClient.api.put(containerName, config)
    }

    public async deleteContainer(containerName: string, config?: AxiosRequestConfig): Promise<void> {
        const hasObjects = !!(await this.getObjectCount(containerName))

        if(hasObjects) {
            const { isEmpty, response } = await this.deleteAllObjects(containerName)

            if(!isEmpty) {
                throw new Error(`Could not remove all objects before deleting the container: ${response}`);
            }
        }

        await SwiftClient.api.delete(containerName, config)
    }
}
