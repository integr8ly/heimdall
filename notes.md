will need to read deployments
will need to read update imagestreams / import-images

accept a namespace and and a list of deployment, pods, dcs it should care about.
check the image in those resources
check did the image come from an image stream
for each image gathered in a ns, check the registery to ensure it is upto date (IE check the manifest  sha against the sha of the image currently being used) 
for none imagestream images we would need to check the pods containerStatuses.imageID