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
            assertTrue("It must contain 2 service", index.getTotal().equals(2));

            System.out.println("Verification registration file : "+System.getenv().getOrDefault(FILENAME, DEFAULT_FILE_NAME));
            File file = new File(System.getenv().getOrDefault(FILENAME, DEFAULT_FILE_NAME));

            if(!file.exists()){
                System.err.println("File do not exist: File must exist in "+DEFAULT_FILE_NAME+" or the environmental variable "+FILENAME+" must be set!");
                System.exit(-1);
            }

            Service service = mapper.readValue(new File(System.getenv().getOrDefault(FILENAME, DEFAULT_FILE_NAME)), Service.class);

            Optional<Service> optional= index.getServices().stream().filter(s->s.getName().equals(service.getName())).findFirst();

            if(!optional.isPresent()) {
                System.err.println("The service "+service.getName()+" was not found in the Service Catalog");
                fail();
            }

            comp(service,optional.get());

        }catch (Exception e){
            System.err.println(e.getMessage());
            fail();
        }

        System.out.println("Registration Integration Test finished");
    }
    private void comp(Service s1, Service s2){

        assertTrue("Name must be equal", s1.getName().equals(s2.getName()));
        assertTrue("Description must be equal", s1.getDescription().equals(s2.getDescription()));
        assertTrue("Docs must be equal", s1.getDocs().equals(s2.getDocs()));
        assertTrue("Apis must be equal", s1.getApis().equals(s2.getApis()));
        assertTrue("Meta must be equal", s1.getMeta().equals(s2.getMeta()));

    }
}
