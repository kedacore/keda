# Weather-Aware Ride Demand Scaler

This scaler allows scaling based on a ride demand metric that can be dynamically adjusted by current weather conditions. It fetches ride demand from one configurable API endpoint and weather conditions from another.

## Configuration Parameters

Here is a list of configuration parameters for the Weather-Aware Ride Demand Scaler:

**Weather API Configuration:**

- **`weatherApiEndpoint`** (Optional): The HTTP(S) endpoint of the weather API.
  - Example: `https://api.weatherprovider.com/v1/current`
- **`weatherApiKeyFromEnv`** (Optional): The name of the environment variable that contains the API key for the weather API. The value from this environment variable will be used for Bearer token authentication.
  - Example: `WEATHER_API_SECRET_KEY`
- **`weatherLocation`** (Optional): The location for which to fetch weather data (e.g., city, country code, or latitude,longitude). The exact format depends on the weather API provider.
  - Example: `London,UK` or `40.7128,-74.0060`
- **`weatherUnits`** (Optional): The units to use for weather data (e.g., "metric" or "imperial"). Defaults to `metric`.
  - Example: `imperial`
- **`badWeatherConditions`** (Optional): A comma-separated list of conditions that define "bad weather". Each condition is a key-value pair separated by a colon. Supported keys are `temp_below`, `temp_above`, `rain_below`, `rain_above`, `wind_below`, `wind_above`. The scaler assumes weather API responses are flat JSON objects with numeric values for keys like "temp", "rain", "wind".
  - Example: `temp_below:0,rain_above:5` (temperature in Celsius below 0 OR rain in mm/hr above 5 constitutes bad weather if `weatherUnits` is `metric`).
  - Example: `temp_below:32,wind_above:15` (temperature in Fahrenheit below 32 OR wind in mph above 15 constitutes bad weather if `weatherUnits` is `imperial`).

**Demand API Configuration:**

- **`demandApiEndpoint`** (Optional): The HTTP(S) endpoint of the ride demand API.
  - Example: `https://api.ridedemand.com/v1/current_demand`
- **`demandApiKeyFromEnv`** (Optional): The name of the environment variable that contains the API key for the demand API. Used for Bearer token authentication.
  - Example: `DEMAND_API_SECRET_KEY`
- **`demandJsonPath`** (Optional): A JSONPath expression to extract the numerical demand value from the demand API's JSON response.
  - Example: `{.data.current_rides}` or `{.demand_level}`. If the response is a simple number, this might not be needed or can be set to extract the root. The scaler's `extractValueWithJSONPath` helper will attempt to use `{.value}` if the path is empty and the response is a map, or convert directly if the response is a number.

**Scaling Logic Configuration:**

- **`targetDemandPerReplica`** (Optional): The target value for ride demand that each replica should handle. Used by the HPA to calculate desired replicas. Defaults to `100`.
  - Example: `50` (meaning if total demand is 200, HPA will aim for 4 replicas).
- **`activationDemandLevel`** (Optional): The threshold of (potentially weather-adjusted) demand above which the scaler becomes active and scales from zero. Defaults to `10`.
  - Example: `20`
- **`weatherEffectScaleFactor`** (Optional): A multiplier applied to the fetched demand if `badWeatherConditions` are met. For example, `1.5` increases the perceived demand by 50%. Defaults to `1.0` (no change).
  - Example: `1.5`
- **`metricName`** (Optional): The name used for the metric in HPA. This will be prefixed with `sX-` where `X` is the trigger index. Defaults to `weather-aware-ride-demand`.
  - Example: `my-custom-city-demand`

**Note on API Keys:**
If `weatherApiKeyFromEnv` or `demandApiKeyFromEnv` are used, ensure the corresponding environment variables are available in your deployment. These are typically set by referencing Kubernetes Secrets or ConfigMaps. The scaler uses these for Bearer token authentication. If your API requires a different authentication method (e.g., query parameters, custom headers), this scaler might need modification or you'd need to use an intermediate proxy.

## Example ScaledObject

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: weather-aware-demand-deployment
  namespace: my-app
spec:
  scaleTargetRef:
    name: my-app-deployment
  pollingInterval: 30   # Optional. Default: 30 seconds
  cooldownPeriod:  300  # Optional. Default: 300 seconds
  minReplicaCount: 0    # Optional. Default: 0
  maxReplicaCount: 10   # Optional. Default: 100
  triggers:
  - type: weather-aware-demand
    metadata:
      # Weather API Details
      weatherApiEndpoint: "https://my-weather-api.com/data"
      weatherApiKeyFromEnv: "WEATHER_API_KEY_ENV_NAME" # Points to an env var in the KEDA operator's deployment or the target deployment
      weatherLocation: "NewYork,US"
      weatherUnits: "imperial"
      badWeatherConditions: "temp_below:32,rain_above:0.2" # Temp in F, Rain in inches/hr (example)

      # Demand API Details
      demandApiEndpoint: "https://my-demand-api.com/rides"
      demandApiKeyFromEnv: "DEMAND_API_KEY_ENV_NAME"
      demandJsonPath: "{.current_demand_value}"

      # Scaling Logic
      targetDemandPerReplica: "25"
      activationDemandLevel: "5"
      weatherEffectScaleFactor: "1.6"
      metricName: "nyc-ride-demand"
```

In this example:
- KEDA will poll the `weather-aware-demand` scaler.
- The scaler will call `https://my-weather-api.com/data` for weather in "NewYork,US" using imperial units and an API key from the `WEATHER_API_KEY_ENV_NAME` environment variable.
- It will also call `https://my-demand-api.com/rides` for demand data, using an API key from `DEMAND_API_KEY_ENV_NAME`, and extract the demand from the `current_demand_value` field in the JSON response.
- If the temperature drops below 32°F or rain is above 0.2 inches/hr, the fetched demand will be multiplied by 1.6.
- The HPA will target 25 (adjusted) demand units per replica.
- The deployment will scale up from 0 if adjusted demand exceeds 5.
- The metric exposed to HPA will be named (e.g.) `s0-nyc-ride-demand`.

Make sure to create the necessary secrets in your namespace that your KEDA operator or target deployment can reference to populate the `WEATHER_API_KEY_ENV_NAME` and `DEMAND_API_KEY_ENV_NAME` environment variables if you use API key authentication.
```
