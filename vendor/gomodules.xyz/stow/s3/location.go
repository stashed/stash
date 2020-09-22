package s3

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
	"gomodules.xyz/stow"
)

// A location contains a client + the configurations used to create the client.
type location struct {
	config         stow.Config
	customEndpoint string
	client         *s3.S3
}

// CreateContainer creates a new container, in this case an S3 bucket.
// The bare minimum needed is a container name, but there are many other
// options that can be provided.
func (l *location) CreateContainer(containerName string) (stow.Container, error) {
	createBucketParams := &s3.CreateBucketInput{
		Bucket: aws.String(containerName), // required
	}

	_, err := l.client.CreateBucket(createBucketParams)
	if err != nil {
		return nil, errors.Wrap(err, "CreateContainer, creating the bucket")
	}

	region, _ := l.config.Config("region")

	newContainer := &container{
		name:           containerName,
		client:         l.client,
		region:         region,
		customEndpoint: l.customEndpoint,
	}

	return newContainer, nil
}

// Containers returns a slice of the Container interface, a cursor, and an error.
// This doesn't seem to exist yet in the API without doing a ton of manual work.
// Get the list of buckets, query every single one to retrieve region info, and finally
// return the list of containers that have a matching region against the client. It's not
// possible to manipulate a container within a region that doesn't match the clients'.
// This is because AWS user credentials can be tied to regions. One solution would be
// to start a new client for every single container where the region matches, this would
// also check the credentials on every new instance... Tabled for later.
func (l *location) Containers(prefix, cursor string, count int) ([]stow.Container, string, error) {
	// Response returns exported Owner(*s3.Owner) and Bucket(*s3.[]Bucket)
	var params *s3.ListBucketsInput
	bucketList, err := l.client.ListBuckets(params)
	if err != nil {
		return nil, "", errors.Wrap(err, "Containers, listing the buckets")
	}

	// Seek to the current bucket, according to cursor.
	if cursor != stow.CursorStart {
		ok := false
		for i, b := range bucketList.Buckets {
			if *b.Name == cursor {
				ok = true
				bucketList.Buckets = bucketList.Buckets[i:]
				break
			}
		}
		if !ok {
			return nil, "", stow.ErrBadCursor
		}
	}
	cursor = ""

	// Region is pulled from stow.Config. If Region is specified, only add
	// Bucket to Container list if it is located in configured Region.
	region, regionSet := l.config.Config(ConfigRegion)

	// Endpoint would indicate that we are using s3-compatible storage, which
	// does not support s3session.GetBucketRegion().
	endpoint, endpointSet := l.config.Config(ConfigEndpoint)

	// Iterate through the slice of pointers to buckets
	var containers []stow.Container
	for _, bucket := range bucketList.Buckets {
		if len(containers) == count {
			cursor = *bucket.Name
			break
		}

		if !strings.HasPrefix(*bucket.Name, prefix) {
			continue
		}

		var err error
		client := l.client
		bucketRegion := region
		if !endpointSet && endpoint == "" {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			bucketRegion, err = s3manager.GetBucketRegionWithClient(ctx, l.client, *bucket.Name)
			cancel()
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
					// sometimes buckets will still show up int eh ListBuckets results after
					// being deleted, but will 404 when determining the region. Use this as a
					// strong signal that the bucket has been deleted.
					continue
				}
				return nil, "", errors.Wrapf(err, "Containers, getting bucket region for: %s", *bucket.Name)
			}
			if regionSet && region != "" && bucketRegion != region {
				continue
			}

			client, _, err = newS3Client(l.config, bucketRegion)
			if err != nil {
				return nil, "", errors.Wrapf(err, "Containers, creating new client for region: %s", bucketRegion)
			}
		}

		newContainer := &container{
			name:           *(bucket.Name),
			client:         client,
			region:         bucketRegion,
			customEndpoint: l.customEndpoint,
		}

		containers = append(containers, newContainer)
	}

	return containers, cursor, nil
}

// Close simply satisfies the Location interface. There's nothing that
// needs to be done in order to satisfy the interface.
func (l *location) Close() error {
	return nil // nothing to close
}

// Container retrieves a stow.Container based on its name which must be
// exact.
func (l *location) Container(id string) (stow.Container, error) {
	client := l.client
	bucketRegion, _ := l.config.Config(ConfigRegion)

	// Endpoint would indicate that we are using s3-compatible storage, which
	// does not support s3session.GetBucketRegion().
	if endpoint, endpointSet := l.config.Config(ConfigEndpoint); !endpointSet && endpoint == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		bucketRegion, _ = s3manager.GetBucketRegionWithClient(ctx, l.client, id)
		cancel()

		var err error
		client, _, err = newS3Client(l.config, bucketRegion)
		if err != nil {
			return nil, errors.Wrapf(err, "Container, creating new client for region: %s", bucketRegion)
		}
	}

	params := &s3.GetBucketLocationInput{
		Bucket: aws.String(id),
	}

	_, err := client.GetBucketLocation(params)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NoSuchBucket" {
			return nil, stow.ErrNotFound
		}

		return nil, errors.Wrap(err, "GetBucketLocation")
	}

	c := &container{
		name:           id,
		client:         client,
		region:         bucketRegion,
		customEndpoint: l.customEndpoint,
	}

	return c, nil
}

// RemoveContainer removes a container simply by name.
func (l *location) RemoveContainer(id string) error {
	params := &s3.DeleteBucketInput{
		Bucket: aws.String(id),
	}

	_, err := l.client.DeleteBucket(params)
	if err != nil {
		return errors.Wrap(err, "RemoveContainer, deleting the bucket")
	}

	return nil
}

// ItemByURL retrieves a stow.Item by parsing the URL, in this
// case an item is an object.
func (l *location) ItemByURL(url *url.URL) (stow.Item, error) {
	if l.customEndpoint == "" {
		genericURL := []string{"https://s3-", ".amazonaws.com/"}

		// Remove genericURL[0] from URL:
		// url = <genericURL[0]><region><genericURL[1]><bucket name><object path>
		firstCut := strings.Replace(url.Path, genericURL[0], "", 1)

		// find first dot so that we could extract region.
		dotIndex := strings.Index(firstCut, ".")

		// region of the s3 bucket.
		region := firstCut[0:dotIndex]

		// Remove <region><genericURL[1]> from
		// <region><genericURL[1]><bucket name><object path>
		secondCut := strings.Replace(firstCut, region+genericURL[1], "", 1)

		// Get the index of the first slash to get the end of the bucket name.
		firstSlash := strings.Index(secondCut, "/")

		// Grab bucket name
		bucketName := secondCut[:firstSlash]

		// Everything afterwards pertains to object.
		objectPath := secondCut[firstSlash+1:]

		// Get the container by bucket name.
		cont, err := l.Container(bucketName)
		if err != nil {
			return nil, errors.Wrapf(err, "ItemByURL, getting container by the bucketname %s", bucketName)
		}

		// Get the item by object name.
		it, err := cont.Item(objectPath)
		if err != nil {
			return nil, errors.Wrapf(err, "ItemByURL, getting item by object name %s", objectPath)
		}

		return it, err
	}

	// url looks like this: s3://<containerName>/<itemName>
	// example: s3://graymeta-demo/DPtest.txt
	containerName := url.Host
	itemName := strings.TrimPrefix(url.Path, "/")

	c, err := l.Container(containerName)
	if err != nil {
		return nil, errors.Wrapf(err, "ItemByURL, getting container by the bucketname %s", containerName)
	}

	i, err := c.Item(itemName)
	if err != nil {
		return nil, errors.Wrapf(err, "ItemByURL, getting item by object name %s", itemName)
	}
	return i, nil
}
