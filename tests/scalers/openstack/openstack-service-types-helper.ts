import axios, { AxiosInstance } from 'axios'

const requestTimeout = 30000
const serviceTypesAuthorityEndpoint = "https://service-types.openstack.org/service-types.json"

const httpClient: AxiosInstance = axios.create({
    timeout: requestTimeout
})

// getServiceTypes retrieves all historical OpenStack Service Types for a given OpenStack project
export default async function getServiceTypes(projectName: string): Promise<string[]> {
	var url = serviceTypesAuthorityEndpoint

    try {
        const getServiceTypes = await httpClient.get(url)

        if (!getServiceTypes.data['primary_service_by_project'][projectName]) {
            throw new Error(`${projectName} is not known as an OpenStack project`)
        }

        const serviceAliases = getServiceTypes.data
            ['primary_service_by_project']
            [projectName]
            ['aliases']

        if (!serviceAliases) {
            const serviceType = getServiceTypes.data
                ['primary_service_by_project']
                [projectName]
                ['service_type']

            if (!serviceType) {
                return []
            }

            return [serviceType]
        }

        return serviceAliases
    } catch (err) {
        throw new Error(`could not retrieve list of OpenStack service types from server: ${err.message}`)
    }
}
