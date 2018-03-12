package eu.linksmart.testing.registration;

import com.fasterxml.jackson.databind.ObjectMapper;
import io.swagger.client.ApiClient;
import io.swagger.client.api.ScApi;
import io.swagger.client.model.APIIndex;
import io.swagger.client.model.Service;
import io.swagger.client.model.ServiceDocs;
import org.junit.Test;

import java.io.File;
import java.math.BigDecimal;
import java.util.HashMap;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;
import java.util.stream.Collectors;
import java.util.stream.Stream;

import static org.junit.Assert.assertTrue;
import static org.junit.Assert.fail;

public class ServiceTesterIT {
    static final String
            BASE_URL = "http://localhost:8082",
            DEFAULT_FILE_NAME = "test/dummy.json";

    static final String
            BASE_URL_PATH = "base_url",
            FILENAME = "filename";
    @Test
    public void registration(){
        System.out.println("Start registration Integration Test");
        ApiClient client = new ApiClient();
        ObjectMapper mapper = new ObjectMapper();

        System.out.println("SC URL: "+System.getenv().getOrDefault(BASE_URL_PATH, BASE_URL));
        client.setBasePath(System.getenv().getOrDefault(BASE_URL_PATH, BASE_URL));
        ScApi api = new ScApi(client);

        try{

            APIIndex index =  api.rootGet(new BigDecimal(1),new BigDecimal(100));
//            assertTrue("It must contain 2 service", index.getTotal().equals(2));

            System.out.println("Verification registration file : "+System.getenv().getOrDefault(FILENAME, DEFAULT_FILE_NAME));
            File file = new File(System.getenv().getOrDefault(FILENAME, DEFAULT_FILE_NAME));

            if(!file.exists()){
                System.err.println("File do not exist: File must exist in "+DEFAULT_FILE_NAME+" or the environmental variable "+FILENAME+" must be set!");
                System.exit(-1);
            }

            Service template = mapper.readValue(new File(System.getenv().getOrDefault(FILENAME, DEFAULT_FILE_NAME)), Service.class);

            Optional<Service> optional= index.getServices().stream().filter(s->s.getName().equals(template.getName())).findFirst();

            if(!optional.isPresent()) {
                System.err.println("The service "+template.getName()+" was not found in the Service Catalog");
                fail();
            }

            cmp(template,optional.get());

        }catch (Exception e){
            System.err.println(e.getMessage());
            fail();
        }

        System.out.println("Registration Integration Test finished");
    }
    private void cmp(Service template, Service s2){

        cmp(s2.getName(), template.getName(),"Name");
        assertTrue("Name must be equal", template.getName().equals(s2.getName()));

        cmp(s2.getDescription(), template.getDescription(),"Description");
        if(s2.getDescription()!=null && template.getDescription()!=null)
            assertTrue("Description must be equal", template.getDescription().equals(s2.getDescription()));

        cmp(s2.getMeta(), template.getMeta(),"Meta");
        if(s2.getMeta()!=null && template.getMeta()!=null)
            assertTrue("Meta must be equal", template.getMeta().equals(s2.getMeta()));

        cmp(s2.getApis(), template.getApis(),"Apis");
        if(s2.getApis()!=null && template.getApis()!=null)
            assertTrue("It must contain all defined apis", s2.getApis().keySet().containsAll(template.getApis().keySet()));

        cmp(s2.getDocs(), template.getDocs(),"Docs");
        if(s2.getDocs()!=null && template.getDocs()!=null)
            for (ServiceDocs docs: template.getDocs())
                assertTrue("The docs description, apis, and type must match ", s2.getDocs().stream().anyMatch(d2->
                                cmp(docs.getDescription(), d2.getDescription(), "Docs.Description") && docs.getDescription().equals(d2.getDescription()) &&
                                cmp(docs.getType(), d2.getType(), "Docs.Type") && docs.getType().equals(d2.getType()) &&
                                (docs.getApis()==null || cmp(docs.getApis(), d2.getApis(), "Docs.Apis") && docs.getApis().equals(d2.getApis())) )
                );

    }
    private boolean cmp(Object o1, Object o2, String propertyName){
        if( (o1==o2 && o1==null ) || ( o1 != o2 && o1 != null))
            return true;

        assertTrue ("One of the"+propertyName+" property is null but the other is not", o1==o2);
        return false;
    }
}
