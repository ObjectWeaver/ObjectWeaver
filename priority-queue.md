this is the general idea of how the priority queue will need to be structured 

1. change the go-sdk library so that there is a priority option which is number where there are shared enums essentially. 

2. create a different worker pool of requests. ie for the batch requests have one worker pool and for the other requests have the other worker poool 

3. for the client sending out the requests alter it so that it works as a queued systemw here the more important messages/requests have a higher priority. Urgent get sent out first in the queue whereas the normal ones will be sent once all the urgent are done. 

4. for the batch worker pool the requests being combined together will be merged into once larger batch requests. This will be sent out every N minutes or at X threshold of parts of the request which has been formed. 

5. For the batch worker pool there will need to be a new thing which keeps track of the requests in local storage (a redis interface implmentaton will be created later) so that if the batching request fails then it can be retried. 

6. there will need to be logic which sends back all the requests to the individual parts of all the in-progress requests for the batching 
    - additionally this generated information/response will need an option to be sent elsewhere ie to a particluar URL etc so that the developers could just extract that information. 

