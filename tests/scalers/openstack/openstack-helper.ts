import axios, { AxiosInstance } from 'axios'

import getServiceTypes from './openstack-service-types-helper'

const requestTimeout = 30000

const keystoneClient: AxiosInstance = axios.create({
    url: '/v3/auth/tokens',
    baseURL: process.env['OS_AUTH_URL'],
    timeout: requestTimeout
})

interface Credentials {
    token: string
    serviceURL: string
}

interface Endpoint {
    id: string
    interface: string
    region_id: string
    url: string
    region: string
}

class OpenstackHelperClient {
    // Creates a client for an OpenStack project with an embedded token
    public static async CreateClient(osServiceName: string, osServiceURL ?: string): Promise<AxiosInstance> {
        try {
            const { token, serviceURL } = await OpenstackHelperClient.getCredentials(osServiceName, osServiceURL);

            const serviceClient = axios.create({
                baseURL: serviceURL,
                headers: {
                    'x-auth-token': token,
                },
                timeout: requestTimeout
            })

            return serviceClient
        } catch (err) {
            console.log(err.message);
        }
    }

    // Retrieves a token using Openstack credentials.
    // It will use the service URL provided by the user. If not provided, it will look for the URL into the Openstack catalog
    private static async getCredentials(osServiceName: string, osServiceURL ?: string): Promise<Credentials> {
        const credentials = <Credentials>{}
        const osRegion = process.env['OS_REGION_NAME']

        try {
            const response = await keystoneClient.request({
                url: keystoneClient.defaults.url,
                method: 'POST',
                data: {
                    "auth": {
                        "identity": {
                            "methods": [
                                "password"
                            ],
                            "password": {
                                "user": {
                                    "id": process.env['OS_USER_ID'],
                                    "password": process.env['OS_PASSWORD']
                                }
                            }
                        },
                        "scope": {
                            "project": {
                                "id": process.env['OS_PROJECT_ID']
                            }
                        }
                    }
                }
            })

            const token = response.headers['x-subject-token']

            if(token) {
                credentials.token = token
            }

            if (osServiceURL) {
                credentials.serviceURL = osServiceURL
            } else {
                const servicesCatalog = response.data.token.catalog
                const serviceTypes = await getServiceTypes(osServiceName)

                for (let i = 0; i < serviceTypes.length; i++) {
                    const osService = servicesCatalog.find(service => service.type === serviceTypes[i])

                    if (osService) {
                        const endpoint: Endpoint = osService.endpoints.find((e: Endpoint) => {
                            if (e.interface === "public") {
                                if (osRegion) {
                                    if (e.region === osRegion) {
                                        return e
                                    }
                                    return undefined
                                }
                                return e
                            }
                        })

                        if (endpoint) {
                            credentials.serviceURL = endpoint.url
                            break
                        }
                    }
                }

                if (!credentials.serviceURL) {
                    throw new Error(`could not retrieve service URL from OpenStack catalog for service '${osServiceName}'`);
                }
            }
        } catch (err) {
            throw new Error(`could not get token or service URL using existing credentials: ${err.message}`);
        }

        return credentials
    }
}

export default OpenstackHelperClient.CreateClient
